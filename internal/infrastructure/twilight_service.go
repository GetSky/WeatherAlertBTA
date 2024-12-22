// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package infrastructure

import (
	"github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/sixdouglas/suncalc"
	"time"
)

var (
	lat, long = 43.649329, 41.426829 // BTA coordinates
)

type twilightService struct {
}

func NewTwilightService() application.TwilightService {
	return &twilightService{}
}

func (n *twilightService) CheckNauticalTwilight() (bool, error) {
	now := time.Now()
	ref := now.Add(-12 * time.Hour) // Using a reference date to correct premature date translation when reaching 00:00
	start, _ := n.calc(ref, suncalc.NauticalDusk)
	end, _ := n.calc(ref.AddDate(0, 0, 1), suncalc.NauticalDawn)

	return now.After(start) && now.Before(end), nil
}

func (n *twilightService) calc(date time.Time, name suncalc.DayTimeName) (time.Time, error) {
	times := suncalc.GetTimes(date, lat, long)
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
