// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package infrastructure

import (
	"fmt"
	. "github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
	"time"
)

var alertTemplate = `🚨 *Alert*

Wind Speed: *%.1f m/s*
Temperature: *%.1f°C*
Update At: %s
`

var windTemplate = `ℹ️ *Update:*

Wind Speed: *%.1f m/s*
_Wind speed is now below the threshold._
Temperature: *%.1f°C*
Update At: %s
`

type telegramNotifyService struct {
	bot          *tgbotapi.BotAPI
	telegramChat int64
	lastChartID  int
	lastData     Weather
}

func NewTelegramNotifyService(bot *tgbotapi.BotAPI, receiverKey string) NotifyService {
	chat, err := strconv.ParseInt(receiverKey, 10, 64)
	if err != nil {
		fmt.Printf("Failed to parse receiverKey: %v\n", err)
		os.Exit(1)
	}

	return &telegramNotifyService{
		bot:          bot,
		telegramChat: chat,
	}
}

func (c *telegramNotifyService) SendUpdate(chart Chart, data Weather) error {
	var cnf tgbotapi.Chattable

	if data.Hazardous != c.lastData.Hazardous || c.lastChartID == 0 {
		cnf = tgbotapi.PhotoConfig{
			BaseFile: tgbotapi.BaseFile{
				BaseChat: tgbotapi.BaseChat{ChatID: c.telegramChat},
				File:     tgbotapi.FilePath(chart.Path),
			},
			ParseMode: tgbotapi.ModeMarkdown,
			Caption:   c.prepareMessageUpdate(data),
		}
	} else {
		cnf = tgbotapi.EditMessageMediaConfig{
			BaseEdit: tgbotapi.BaseEdit{
				MessageID: c.lastChartID,
				ChatID:    c.telegramChat,
			},
			Media: tgbotapi.InputMediaPhoto{
				BaseInputMedia: tgbotapi.BaseInputMedia{
					Type:      "photo",
					Media:     tgbotapi.FilePath(chart.Path),
					ParseMode: tgbotapi.ModeMarkdown,
					Caption:   c.prepareMessageUpdate(data),
				},
			},
		}
	}

	msg, err := c.bot.Send(cnf)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v", err)
	}
	c.lastChartID = msg.MessageID

	return nil
}

func (c *telegramNotifyService) prepareMessageUpdate(data Weather) string {
	var template string
	if data.Hazardous {
		template = alertTemplate
	} else {
		template = windTemplate
	}

	return fmt.Sprintf(template, data.WindSpeed, data.Temperature, data.UpdateAt.Format(time.TimeOnly))
}

func (c *telegramNotifyService) SendWorkStarted(chart Chart, data Weather) error {
	_, err := c.bot.Send(tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: c.telegramChat,
		},
		Text: "Start.",
	})

	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}

	return nil
}

func (c *telegramNotifyService) SendWorkEnded(chart Chart, data Weather) error {
	_, err := c.bot.Send(tgbotapi.MessageConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: c.telegramChat,
		},
		Text: "End.",
	})

	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}

	return nil
}
