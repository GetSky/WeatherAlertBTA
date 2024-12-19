package application

import "time"

type NotifyService interface {
	SendNewMessage(text string) error
	UpdateLastMessage(text string) error

	SendNewChart(chart Chart, text string) error
	UpdateLastChart(chart Chart, text string) error
}

type ChartService interface {
	GetUpdatedChart() (Chart, error)
}

type Chart struct {
	Path     string
	CreateAt time.Time
}
