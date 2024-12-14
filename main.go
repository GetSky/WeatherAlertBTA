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
	url                string
	botToken           string
	telegramChat       int64
	windThreshold      float64
	DelayTimeInMinutes time.Duration
	pollInterval       time.Duration
	chartWeatherURL    string
)

var (
	lastModified      string
	windAlertActive   = false
	lastMessageID     int
	lastWindAlertTime time.Time
)

func init() {
	url = os.Getenv("WEATHER_URL")
	if url == "" {
		url = "https://www.sao.ru/tb/tcs/meteo/data/meteo.dat"
	}

	chartWeatherURL = os.Getenv("CHART_WEATHER_URL")
	if chartWeatherURL == "" {
		url = "https://relay.sao.ru/tb/tcs/meteo/data/meteo_today.png"
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
		windThreshold = 2.2
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
		DelayTimeInMinutes = 20 * time.Minute
	} else {
		DelayTimeInMinutes, err = time.ParseDuration(intervalStr)
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

func checkWeather(bot *tgbotapi.BotAPI) {
	// Checking for an updated file
	client := &http.Client{}
	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		fmt.Printf("Failed to create HEAD request: %v\n", err)
		return
	}

	headResp, err := client.Do(headReq)
	if err != nil {
		fmt.Printf("HEAD request failed: %v\n", err)
		return
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected HEAD response: %s\n", headResp.Status)
		return
	}

	currentModified := headResp.Header.Get("Last-Modified")
	if currentModified == "" {
		fmt.Println("No Last-Modified header found.")
		return
	}

	if currentModified == lastModified {
		fmt.Println("Data has not changed since last check.")
		return
	}

	lastModified = currentModified

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
			windAlertActive = true

			alertMessage := fmt.Sprintf(alertTemplate, timestamp, temp, windSpeed)
			message := tgbotapi.NewMessage(telegramChat, alertMessage)
			_, err := bot.Send(message)
			if err != nil {
				fmt.Printf("Failed to send message: %v\n", err)
				return
			}

			photo := tgbotapi.NewPhoto(telegramChat, tgbotapi.FileURL(chartWeatherURL))
			_, err = bot.Send(photo)
			if err != nil {
				fmt.Printf("Failed to send image: %v\n", err)
				return
			}

			lastMessageID = 0
			fmt.Println("Wind alert sent successfully.")
		} else {
			fmt.Println("Wind alert already active. No message sent.")
		}
	} else {
		if windAlertActive {
			duration := time.Since(lastWindAlertTime)
			if duration > DelayTimeInMinutes*time.Minute {
				windAlertActive = false

				photo := tgbotapi.NewPhoto(telegramChat, tgbotapi.FileURL(chartWeatherURL))
				_, err := bot.Send(photo)
				if err != nil {
					fmt.Printf("Failed to send image: %v\n", err)
					return
				}

				alertMessage := fmt.Sprintf(windTemplate, timestamp, temp, windSpeed)
				message := tgbotapi.NewMessage(telegramChat, alertMessage)
				msg, err := bot.Send(message)
				if err != nil {
					fmt.Printf("Failed to send message: %v\n", err)
					return
				}

				lastMessageID = msg.MessageID
				fmt.Println("Wind speed below threshold message sent.")
			} else {
				fmt.Println("Wind speed below threshold but duration < 20 minutes. No alert reset.")
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