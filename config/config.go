// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package config

import (
	"fmt"
	"github.com/caarlos0/env/v11"
	"time"
)

type Conf struct {
	WeatherUrl      string  `env:"WEATHER_URL" envDefault:"https://relay.sao.ru/tb/tcs/meteo/data/meteo.dat"`
	ChartWeatherUrl string  `env:"WEATHER_URL" envDefault:"https://www.sao.ru/tb/tcs/meteo/meteo_today.cgi"`
	BotToken        string  `env:"BOT_TOKEN,required"`
	TelegramChat    string  `env:"TELEGRAM_CHAT_ID,required"`
	WindThreshold   float64 `env:"WIND_THRESHOLD" envDefault:"14.5"`

	PollInterval          time.Duration `env:"POLL_INTERVAL" envDefault:"1m"`
	DelayTime             time.Duration `env:"DELAY_TIME_IN_MINUTES" envDefault:"20m"`
	TimeReserveBeforeDusk time.Duration `env:"RESERVE_TIME_BEFORE_DUSK_IN_MINUTES" envDefault:"120m"`
}

func NewConf() (*Conf, error) {
	cnf := &Conf{}
	if err := env.Parse(cnf); err != nil {
		fmt.Printf("error on parse env config: %v", err)
		return nil, err
	}

	return cnf, nil
}
