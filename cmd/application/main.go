// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package main

import (
	"fmt"
	"github.com/GetSky/WeatherAlertBTA/config"
	"github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/GetSky/WeatherAlertBTA/internal/infrastructure"
	"time"
)

var cnf *config.Conf

var weatherSrv application.WeatherService
var chartSrv application.ChartService
var notifySrv application.NotifyService
var scheduleSrv application.ScheduleService

var (
	workActive        bool
	windAlertActive   bool
	lastWindAlertTime time.Time
)

func init() {
	var err error
	cnf, err = config.NewConf()
	if err != nil {
		return
	}

	chartSrv = infrastructure.NewChartService(cnf.ChartWeatherUrl)
	weatherSrv = infrastructure.NewWeatherService(cnf.WeatherUrl)
	notifySrv = infrastructure.NewTelegramNotifyService(cnf.BotToken, cnf.TelegramChat)
	scheduleSrv = infrastructure.NewScheduleService(cnf.TimeReserveBeforeDusk)
}

func main() {
	for {
		isWorkTime, _ := scheduleSrv.IsWorkNow()
		if isWorkTime != workActive {
			workActive = isWorkTime
			checkWorkStatus()
		}
		if isWorkTime {
			checkWeather()
		} else {
		}
		time.Sleep(cnf.PollInterval)
	}
}

func checkWorkStatus() {
	switch workActive {
	case true:
		dusk, dawn, err := scheduleSrv.GetNautical(time.Now())
		if err != nil {
			fmt.Printf("Main → %v\n", err)
			return
		}
		err = notifySrv.SendWorkStarted(dusk, dawn)
		if err != nil {
			fmt.Printf("Main → %v\n", err)
			return
		}
	case false:
		err := notifySrv.SendWorkEnded()
		if err != nil {
			fmt.Printf("Main → %v\n", err)
			return
		}
	}
}

func checkWeather() {
	chart, err := chartSrv.GetUpdatedChart()
	if err != nil {
		fmt.Printf("ChartService → %v\n", err)
		return
	}

	weather, err := weatherSrv.GetLatestWeather()
	if err != nil {
		fmt.Printf("WeatherService → %v\n", err)
		return
	}

	if weather.WindSpeed >= cnf.WindThreshold {
		weather.Hazardous = true
		lastWindAlertTime = time.Now()
		windAlertActive = true

		err = notifySrv.SendUpdate(chart, weather)
		if err != nil {
			fmt.Printf("Main → %v\n", err)
			return
		}

		fmt.Println("Wind alert sent successfully.")

	} else {
		if windAlertActive {
			duration := time.Since(lastWindAlertTime)
			if duration > cnf.DelayTime {
				weather.Hazardous = false
				err = notifySrv.SendUpdate(chart, weather)
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
			weather.Hazardous = false
			err = notifySrv.SendUpdate(chart, weather)
			if err != nil {
				fmt.Printf("Main → %v\n", err)
				return
			}
		}
	}
}
