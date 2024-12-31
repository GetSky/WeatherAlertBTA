// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package application

import "time"

type NotifyService interface {
	SendWorkStarted(chart Chart, data Weather) error
	SendWorkEnded(chart Chart, data Weather) error
	SendUpdate(chart Chart, data Weather) error
}

type ScheduleService interface {
	IsWorkNow() (bool, error)
	GetNautical(date time.Time) (dusk time.Time, dawn time.Time, err error)
}

type ChartService interface {
	GetUpdatedChart() (Chart, error)
}

type Chart struct {
	Path     string
	CreateAt time.Time
}

type Weather struct {
	WindSpeed   float64
	Temperature float64
	Hazardous   bool
	UpdateAt    time.Time
}
