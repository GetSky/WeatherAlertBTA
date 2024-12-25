// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package infrastructure

import (
	"fmt"
	. "github.com/GetSky/WeatherAlertBTA/internal/application"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
)

type telegramNotifyService struct {
	bot           *tgbotapi.BotAPI
	telegramChat  int64
	lastChartID   int
	lastMessageID int
	lastMsgText   string
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

func (c *telegramNotifyService) SendNewMessage(text string) error {
	message := tgbotapi.NewMessage(c.telegramChat, text)
	message.ParseMode = tgbotapi.ModeMarkdown

	msg, err := c.bot.Send(message)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}
	c.lastMessageID = msg.MessageID

	return nil
}

func (c *telegramNotifyService) UpdateLastMessage(text string) error {
	if c.lastMessageID == 0 {
		return c.SendNewMessage(text)
	}

	editedMessage := tgbotapi.NewEditMessageText(
		c.telegramChat,
		c.lastMessageID,
		text,
	)
	editedMessage.ParseMode = tgbotapi.ModeMarkdown

	_, err := c.bot.Send(editedMessage)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}

	return nil
}

func (c *telegramNotifyService) SendNewChart(chart Chart, text string) error {
	cnf := tgbotapi.NewPhoto(c.telegramChat, tgbotapi.FilePath(chart.Path))
	cnf.ParseMode = tgbotapi.ModeMarkdown
	if text != "" {
		cnf.Caption = text
		c.lastMsgText = text
	} else {
		cnf.Caption = c.lastMsgText
	}

	msg, err := c.bot.Send(cnf)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v", err)
	}
	c.lastChartID = msg.MessageID

	return nil
}

func (c *telegramNotifyService) UpdateLastChart(chart Chart, text string) error {
	if c.lastChartID == 0 {
		return nil
	}

	cnf := tgbotapi.NewInputMediaPhoto(tgbotapi.FilePath(chart.Path))
	cnf.ParseMode = tgbotapi.ModeMarkdown
	if text != "" {
		cnf.Caption = text
		c.lastMsgText = text
	} else {
		cnf.Caption = c.lastMsgText
	}

	_, err := c.bot.Send(tgbotapi.EditMessageMediaConfig{
		BaseEdit: tgbotapi.BaseEdit{
			MessageID: c.lastChartID,
			ChatID:    c.telegramChat,
		},
		Media: cnf,
	})

	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v", err)
	}

	return nil
}
