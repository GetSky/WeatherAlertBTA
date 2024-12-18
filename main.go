package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type NotifyService interface {
	SendNewMessage(text string) error
	UpdateLastMessage(text string) error

	SendNewChart(chart Chart, text string) error
	UpdateLastChart(chart Chart, text string) error
}

type ChartService interface {
	GetUpdatedChart() (Chart, error)
}

type Chart struct {
	Path     string
	CreateAt time.Time
}

var alertTemplate = `🚨 Wind Alert:

Date: %s
Temperature: %s°C
Wind Speed: %.1f m/s
`

var windTemplate = `ℹ️ Wind Update:

Date: %s
Temperature: %s°C
Wind Speed: %.1f m/s
Wind speed is now below the threshold.`

var (
	url             string
	botToken        string
	telegramChat    string
	windThreshold   float64
	DelayTime       time.Duration
	pollInterval    time.Duration
	chartWeatherURL string
)

var client *http.Client
var chartSrv ChartService
var notifySrv NotifyService

var (
	lastModified      string
	windAlertActive   bool
	lastWindAlertTime time.Time
)

func init() {
	client = &http.Client{}

	url = os.Getenv("WEATHER_URL")
	if url == "" {
		url = "https://relay.sao.ru/tb/tcs/meteo/data/meteo.dat"
	}

	chartWeatherURL = os.Getenv("CHART_WEATHER_URL")
	if chartWeatherURL == "" {
		chartWeatherURL = "https://www.sao.ru/tb/tcs/meteo/meteo_today.cgi"
	}

	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		fmt.Println("BOT_TOKEN is not set")
		os.Exit(1)
	}

	telegramChat = os.Getenv("TELEGRAM_CHAT_ID")
	if telegramChat == "" {
		fmt.Println("TELEGRAM_CHAT_ID is not set")
		os.Exit(1)
	}

	var err error

	thresholdStr := os.Getenv("WIND_THRESHOLD")
	if thresholdStr == "" {
		windThreshold = 14.5
	} else {
		windThreshold, err = strconv.ParseFloat(thresholdStr, 64)
		if err != nil {
			fmt.Printf("Failed to parse WIND_THRESHOLD: %v\n", err)
			os.Exit(1)
		}
	}

	intervalStr := os.Getenv("POLL_INTERVAL")
	if intervalStr == "" {
		pollInterval = 1 * time.Minute
	} else {
		pollInterval, err = time.ParseDuration(intervalStr)
		if err != nil {
			fmt.Printf("Failed to parse POLL_INTERVAL: %v\n", err)
			os.Exit(1)
		}
	}

	delayTimeInMinutesStr := os.Getenv("DELAY_TIME_IN_MINUTES")
	if delayTimeInMinutesStr == "" {
		DelayTime = 20 * time.Minute
	} else {
		DelayTime, err = time.ParseDuration(delayTimeInMinutesStr)
		if err != nil {
			fmt.Printf("Failed to parse DELAY_TIME_IN_MINUTES: %v\n", err)
			os.Exit(1)
		}
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		fmt.Printf("Failed to initialize bot: %v\n", err)
		os.Exit(1)
	}

	chartSrv = NewChartService(chartWeatherURL)
	notifySrv = NewTelegramNotifyService(bot, telegramChat)

	for {
		checkWeather()
		time.Sleep(pollInterval)
	}
}

func GetLastUpdate(url string) (string, error) {
	modifiedAt := ""
	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return modifiedAt, fmt.Errorf("Failed to create HEAD request: %v\n", err)
	}

	headResp, err := client.Do(headReq)
	if err != nil {
		return modifiedAt, fmt.Errorf("HEAD request failed: %v\n", err)
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return modifiedAt, fmt.Errorf("Unexpected HEAD response: %s\n", headResp.Status)
	}

	modifiedAt = headResp.Header.Get("Last-Modified")
	return modifiedAt, nil
}

func checkWeather() {
	// Checking for an updated file
	modifiedAt, err := GetLastUpdate(url)
	if err != nil {
		fmt.Printf("Failed to check last update: %v\n", err)
		return
	}

	if windAlertActive {
		chart, err := chartSrv.GetUpdatedChart()
		if err != nil {
			fmt.Printf("ChartService → %v\n", err)
		} else {
			err := notifySrv.UpdateLastChart(chart, "")
			if err != nil {
				fmt.Printf("Main → %v\n", err)
			}
		}
	}

	if modifiedAt == lastModified {
		fmt.Println("Data has not changed since last check.")
		return
	}

	lastModified = modifiedAt

	// Use Range header to fetch only the last 66 bytes (approximately one line)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Failed to create GET request: %v\n", err)
		return
	}
	req.Header.Set("Range", "bytes=-66")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch weather data: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		fmt.Printf("Unexpected HTTP response: %s\n", resp.Status)
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan()
	var lastLine = scanner.Text()

	if lastLine == "" {
		fmt.Println("No data available.")
		return
	}

	fields := strings.Fields(lastLine)
	if len(fields) < 9 {
		fmt.Println("Malformed last line.")
		return
	}

	windSpeed, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		fmt.Printf("Error parsing wind speed: %v\n", err)
		return
	}

	temp := fields[3]
	timestamp := fmt.Sprintf("%s %s", fields[0], fields[1])

	if windSpeed >= windThreshold {
		lastWindAlertTime = time.Now()
		if !windAlertActive {

			chart, err := chartSrv.GetUpdatedChart()
			if err != nil {
				fmt.Printf("ChartService → %v\n", err)
				return
			}

			err = notifySrv.SendNewMessage(fmt.Sprintf(alertTemplate, timestamp, temp, windSpeed))
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}

			err = notifySrv.SendNewChart(chart, "")
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}

			windAlertActive = true
			fmt.Println("Wind alert sent successfully.")
		} else {
			fmt.Println("Wind alert already active. No message sent.")
		}
	} else {
		if windAlertActive {
			duration := time.Since(lastWindAlertTime)
			if duration > DelayTime {
				chart, err := chartSrv.GetUpdatedChart()
				if err != nil {
					fmt.Printf("CharrtService → %v\n", err)
				} else {
					err := notifySrv.UpdateLastChart(chart, "")
					if err != nil {
						fmt.Printf("Main → %v\n", err)
					}
				}

				err = notifySrv.SendNewMessage(fmt.Sprintf(windTemplate, timestamp, temp, windSpeed))
				if err != nil {
					fmt.Printf("Main → %v\n", err)
					return
				}

				windAlertActive = false
				fmt.Println("Wind speed below threshold message sent.")
			} else {
				fmt.Println("The wind speed is below the threshold, but the time has not come to cancel the alert.")
			}
		} else {
			err = notifySrv.SendNewMessage(fmt.Sprintf(windTemplate, timestamp, temp, windSpeed))
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}
		}
	}
}
