// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package main

import (
	"bufio"
	"fmt"
	"github.com/GetSky/WeatherAlertBTA/config"
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

var cnf *config.Conf

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

	var err error
	cnf, err = config.NewConf()
	if err != nil {
		return
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(cnf.BotToken)
	if err != nil {
		fmt.Printf("Failed to initialize bot: %v\n", err)
		os.Exit(1)
	}
	chartSrv = NewChartService(cnf.ChartWeatherUrl)
	notifySrv = NewTelegramNotifyService(bot, cnf.TelegramChat)
	twilightSrv = NewTwilightService(cnf.TimeReserveBeforeDusk)

	for {
		isWorkTime, _ := twilightSrv.IsWorkNow()
		if isWorkTime {
			checkWeather()
		}
		time.Sleep(cnf.PollInterval)
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
	modifiedAt, err := GetLastUpdate(cnf.WeatherUrl)
	if err != nil {
		fmt.Printf("Failed to check last update: %v\n", err)
		return
	}

	chart, err := chartSrv.GetUpdatedChart()
	if err != nil {
		fmt.Printf("ChartService → %v\n", err)
		return
	}

	/* ToDo: Return logic after transfer to weather data receiving service
	err = notifySrv.UpdateLastChart(chart, "")
	if err != nil {
		fmt.Printf("Main → %v\n", err)
	}
	*/

	if modifiedAt == lastModified {
		fmt.Println("Data has not changed since last check.")
		return
	}

	lastModified = modifiedAt

	// Use Range header to fetch only the last 66 bytes (approximately one line)
	req, err := http.NewRequest("GET", cnf.WeatherUrl, nil)
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

	temp, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		fmt.Printf("Error parsing temperature: %v\n", err)
		return
	}

	timestamp := fmt.Sprintf("%s %s", fields[0], fields[1])
	updateAt, err := time.Parse("02-Jan-2006 15:04:05", timestamp)
	if err != nil {
		fmt.Printf("Error parsing date time: %v\n", err)
		return
	}

	data := Weather{
		UpdateAt:    updateAt,
		Temperature: temp,
		WindSpeed:   windSpeed,
	}

	if windSpeed >= cnf.WindThreshold {
		data.Hazardous = true
		lastWindAlertTime = time.Now()
		windAlertActive = true

		err = notifySrv.SendUpdate(chart, data)
		if err != nil {
			fmt.Printf("Main → %v\n", err)
			return
		}

		fmt.Println("Wind alert sent successfully.")

	} else {
		if windAlertActive {
			duration := time.Since(lastWindAlertTime)
			if duration > cnf.DelayTime {

				data.Hazardous = false
				err = notifySrv.SendUpdate(chart, data)
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
			data.Hazardous = false
			err = notifySrv.SendUpdate(chart, data)
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}
		}
	}
}
