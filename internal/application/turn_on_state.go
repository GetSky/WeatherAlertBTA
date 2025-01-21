// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package application

import (
	"fmt"
	"time"
)

type TurnOnState struct {
	tracker     *WeatherTracker
	scheduleSrv ScheduleService
	notifySrv   NotifyService
	chartSrv    ChartService
	weatherSrv  WeatherService

	windThreshold float64
	delayTime     time.Duration

	lastWindAlertTime time.Time
	windAlertActive   bool
}

func NewTurnOnState(
	schedule ScheduleService,
	notify NotifyService,
	chart ChartService,
	weather WeatherService,
	windThreshold float64,
	delayTime time.Duration,
) *TurnOnState {
	return &TurnOnState{
		scheduleSrv:   schedule,
		notifySrv:     notify,
		chartSrv:      chart,
		weatherSrv:    weather,
		windThreshold: windThreshold,
		delayTime:     delayTime,
	}
}

func (t *TurnOnState) SetTracker(tracker *WeatherTracker) {
	t.tracker = tracker
}

func (t *TurnOnState) check() error {
	isWorkTime, err := t.scheduleSrv.IsWorkNow()
	if err != nil {
		return err
	}

	if isWorkTime == false {
		err = t.notifySrv.SendWorkEnded()
		if err != nil {
			return err
		}

		t.tracker.switchState(t.tracker.turnedOff)

		return nil
	}

	err = t.checkWeather()
	if err != nil {
		return err
	}

	return nil
}

func (t *TurnOnState) checkWeather() error {
	chart, err := t.chartSrv.GetUpdatedChart()
	if err != nil {
		return err
	}

	weather, err := t.weatherSrv.GetLatestWeather()
	if err != nil {
		return err
	}

	if weather.WindSpeed >= t.windThreshold {
		weather.Hazardous = true
		t.lastWindAlertTime = time.Now()
		t.windAlertActive = true

		err = t.notifySrv.SendUpdate(chart, weather)
		if err != nil {
			return err
		}

		fmt.Println("Wind alert sent successfully.")

	} else {
		if t.windAlertActive {
			duration := time.Since(t.lastWindAlertTime)
			if duration > t.delayTime {
				weather.Hazardous = false
				err = t.notifySrv.SendUpdate(chart, weather)
				if err != nil {
					return err
				}

				t.windAlertActive = false
				fmt.Println("Wind speed below threshold message sent.")
			} else {
				fmt.Println("The wind speed is below the threshold, but the time has not come to cancel the alert.")
			}
		} else {
			weather.Hazardous = false
			err = t.notifySrv.SendUpdate(chart, weather)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
