// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package infrastructure

import (
	"fmt"
	"github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/sixdouglas/suncalc"
	"time"
)

var (
	lat, long, height = 43.649329, 41.426829, 2070.0 // BTA coordinates
)

type scheduleService struct {
	beforeDusk time.Duration
}

func NewScheduleService(beforeDusk time.Duration) application.ScheduleService {
	return &scheduleService{
		beforeDusk: beforeDusk,
	}
}

func (n *scheduleService) IsWorkNow() (bool, error) {
	now := time.Now()
	start, end, err := n.GetNautical(now)
	if err != nil {
		return false, err
	}
	start = start.Add(-n.beforeDusk)

	return now.After(start) && now.Before(end), nil
}

func (n *scheduleService) GetNautical(now time.Time) (dusk time.Time, dawn time.Time, err error) {
	err = nil
	ref := now.Add(-12 * time.Hour) // Using a reference date to correct premature date translation when reaching 00:00
	dusk, err = n.calc(ref, suncalc.NauticalDusk)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("scheduleService → %s", err)
	}

	dawn, err = n.calc(ref.AddDate(0, 0, 1), suncalc.NauticalDawn)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("scheduleService → %s", err)
	}

	return dusk, dawn, err
}

func (n *scheduleService) calc(date time.Time, name suncalc.DayTimeName) (time.Time, error) {
	times := suncalc.GetTimesWithObserver(
		date,
		suncalc.Observer{Latitude: lat, Longitude: long, Height: height, Location: time.UTC},
	)

	date = time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		times[name].Value.Hour(),
		times[name].Value.Minute(),
		0,
		0,
		time.UTC,
	)
	return date, nil
}
