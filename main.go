package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	telegramChat    int64
	windThreshold   float64
	DelayTime       time.Duration
	pollInterval    time.Duration
	chartWeatherURL string
)

var client *http.Client

var (
	lastModified             string
	lastModifiedChartWeather string
	windAlertActive          bool
	lastMessageID            int
	lastWindAlertTime        time.Time
)

func init() {
	client = &http.Client{}

	url = os.Getenv("WEATHER_URL")
	if url == "" {
		url = "https://relay.sao.ru/tb/tcs/meteo/data/meteo.dat"
	}

	chartWeatherURL = os.Getenv("CHART_WEATHER_URL")
	if chartWeatherURL == "" {
		chartWeatherURL = "https://relay.sao.ru/tb/tcs/meteo/data/meteo_today.png"
	}

	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		fmt.Println("BOT_TOKEN is not set")
		os.Exit(1)
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	if chatIDStr == "" {
		fmt.Println("TELEGRAM_CHAT_ID is not set")
		os.Exit(1)
	}
	var err error
	telegramChat, err = strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		fmt.Printf("Failed to parse TELEGRAM_CHAT_ID: %v\n", err)
		os.Exit(1)
	}

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

	for {
		checkWeather(bot)
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

func downloadChart() (string, error) {
	pathFile := "chart.png"

	modifiedAt, err := GetLastUpdate(chartWeatherURL)
	if err != nil {
		fmt.Printf("Failed to check last update: %v\n", err)
	}

	if modifiedAt == lastModifiedChartWeather {
		return "", fmt.Errorf("\"Last-Modified\" header has not changed for the file")
	}

	resp, err := http.Get(chartWeatherURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	file, err := os.Create(pathFile)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %v", err)
	}

	lastModifiedChartWeather = modifiedAt

	return pathFile, nil
}

func checkWeather(bot *tgbotapi.BotAPI) {
	// Checking for an updated file
	modifiedAt, err := GetLastUpdate(url)
	if err != nil {
		fmt.Printf("Failed to check last update: %v\n", err)
		return
	}

	if windAlertActive {
		path, err := downloadChart()
		if err != nil {
			fmt.Printf("Failed to download chart: %v\n", err)
		} else {
			_, err := bot.Send(tgbotapi.EditMessageMediaConfig{
				BaseEdit: tgbotapi.BaseEdit{
					MessageID: lastMessageID,
					ChatID:    telegramChat,
				},
				Media: tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path)),
			})
			if err != nil {
				fmt.Printf("Failed to send active alert image: %v\n", err)
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

			path, err := downloadChart()
			if err != nil {
				fmt.Printf("Failed to download chart: %v\n", err)
				return
			}

			message := tgbotapi.NewMessage(telegramChat, fmt.Sprintf(alertTemplate, timestamp, temp, windSpeed))
			_, err = bot.Send(message)
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
				return
			}

			photo := tgbotapi.NewPhoto(telegramChat, tgbotapi.FilePath(path))
			msg, err := bot.Send(photo)
			if err != nil {
				fmt.Printf("Failed to send active alert image: %v\n", err)
				return
			}
			lastMessageID = msg.MessageID
			windAlertActive = true
			fmt.Println("Wind alert sent successfully.")
		} else {
			fmt.Println("Wind alert already active. No message sent.")
		}
	} else {
		if windAlertActive {
			duration := time.Since(lastWindAlertTime)
			if duration > DelayTime {
				path, err := downloadChart()
				if err != nil {
					fmt.Printf("Failed to download chart: %v\n", err)
					return
				} else {
					_, err := bot.Send(tgbotapi.EditMessageMediaConfig{
						BaseEdit: tgbotapi.BaseEdit{
							MessageID: lastMessageID,
							ChatID:    telegramChat,
						},
						Media: tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(path)),
					})
					if err != nil {
						fmt.Printf("Failed to send active alert image: %v\n", err)
					}
				}

				message := tgbotapi.NewMessage(telegramChat, fmt.Sprintf(windTemplate, timestamp, temp, windSpeed))
				msg, err := bot.Send(message)
				if err != nil {
					fmt.Printf("Failed to send message: %v\n", err)
					return
				}

				windAlertActive = false
				lastMessageID = msg.MessageID
				fmt.Println("Wind speed below threshold message sent.")
			} else {
				fmt.Println("The wind speed is below the threshold, but the time has not come to cancel the alert.")
			}
		} else {
			if lastMessageID != 0 {
				editedMessage := tgbotapi.NewEditMessageText(
					telegramChat,
					lastMessageID,
					fmt.Sprintf(windTemplate, timestamp, temp, windSpeed),
				)
				_, err = bot.Send(editedMessage)
				if err != nil {
					fmt.Printf("Failed to edit message: %v\n", err)
					return
				}
				fmt.Println("Updated wind information.")
			} else {
				fmt.Println("No previous message to update.")
			}
		}
	}
}
