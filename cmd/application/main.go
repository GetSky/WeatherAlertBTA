// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package main

import (
	"github.com/GetSky/WeatherAlertBTA/config"
	"github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/GetSky/WeatherAlertBTA/internal/infrastructure"
	"time"
)

var cnf *config.Conf
var tracker *application.WeatherTracker

func init() {
	var err error
	cnf, err = config.NewConf()
	if err != nil {
		return
	}

	chartSrv := infrastructure.NewChartService(cnf.ChartWeatherUrl)
	weatherSrv := infrastructure.NewWeatherService(cnf.WeatherUrl)
	notifySrv := infrastructure.NewTelegramNotifyService(cnf.BotToken, cnf.TelegramChat)
	scheduleSrv := infrastructure.NewScheduleService(cnf.TimeReserveBeforeDusk)

	tracker = application.NewWeatherTracker(
		application.NewTurnOnState(scheduleSrv, notifySrv, chartSrv, weatherSrv, cnf.WindThreshold, cnf.DelayTime),
		application.NewTurnOffState(scheduleSrv, notifySrv),
	)

}

func main() {
	for {
		tracker.Check()
		time.Sleep(cnf.PollInterval)
	}
}
