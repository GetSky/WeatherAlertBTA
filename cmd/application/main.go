// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package main

import (
	"bufio"
	"fmt"
	. "github.com/GetSky/WeatherAlertBTA/internal/application"
	. "github.com/GetSky/WeatherAlertBTA/internal/infrastructure"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var alertTemplate = `🚨 *Wind Alert* %s

Temperature: *%s°C*
Wind Speed: *%.1f m/s*
`

var windTemplate = `ℹ️ *Wind Update* %s

Temperature: *%s°C*
Wind Speed: *%.1f m/s*
_Wind speed is now below the threshold._`

var (
	url                   string
	botToken              string
	telegramChat          string
	windThreshold         float64
	DelayTime             time.Duration
	pollInterval          time.Duration
	chartWeatherURL       string
	timeReserveBeforeDusk time.Duration
)

var client *http.Client
var chartSrv ChartService
var notifySrv NotifyService
var twilightSrv ScheduleService

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

	timeReserveBeforeDuskStr := os.Getenv("RESERVE_TIME_BEFORE_DUSK_IN_MINUTES")
	if timeReserveBeforeDuskStr == "" {
		timeReserveBeforeDusk = 120 * time.Minute
	} else {
		timeReserveBeforeDusk, err = time.ParseDuration(timeReserveBeforeDuskStr)
		if err != nil {
			fmt.Printf("Failed to parse RESERVE_TIME_BEFORE_DUSK_IN_MINUTES: %v\n", err)
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
	twilightSrv = NewTwilightService(timeReserveBeforeDusk)

	for {
		isWorkTime, _ := twilightSrv.IsWorkNow()
		if isWorkTime {
			checkWeather()
		}
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

	chart, err := chartSrv.GetUpdatedChart()
	if err != nil {
		fmt.Printf("ChartService → %v\n", err)
		return
	}

	err = notifySrv.UpdateLastChart(chart, "")
	if err != nil {
		fmt.Printf("Main → %v\n", err)

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
			err = notifySrv.SendNewChart(chart, fmt.Sprintf(alertTemplate, timestamp, temp, windSpeed))
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

				err = notifySrv.SendNewChart(chart, fmt.Sprintf(windTemplate, timestamp, temp, windSpeed))
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
			err = notifySrv.UpdateLastChart(chart, fmt.Sprintf(windTemplate, timestamp, temp, windSpeed))
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}
		}
	}
}
