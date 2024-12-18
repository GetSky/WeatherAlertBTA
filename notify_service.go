package main

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"os"
	"strconv"
)

type telegramNotifyService struct {
	bot           *tgbotapi.BotAPI
	telegramChat  int64
	lastChartID   int
	lastMessageID int
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

func (c telegramNotifyService) SendNewMessage(text string) error {
	message := tgbotapi.NewMessage(c.telegramChat, text)
	msg, err := c.bot.Send(message)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}
	c.lastMessageID = msg.MessageID

	return nil
}

func (c telegramNotifyService) UpdateLastMessage(text string) error {
	if c.lastMessageID == 0 {
		return nil
	}

	editedMessage := tgbotapi.NewEditMessageText(
		c.telegramChat,
		c.lastMessageID,
		text,
	)
	_, err := c.bot.Send(editedMessage)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v\n", err)
	}

	return nil
}

func (c telegramNotifyService) SendNewChart(chart Chart, text string) error {
	cnf := tgbotapi.NewPhoto(c.telegramChat, tgbotapi.FilePath(chart.Path))
	if text != "" {
		cnf.Caption = text
	}

	msg, err := c.bot.Send(cnf)
	if err != nil {
		return fmt.Errorf("TelegramNotifyService → %v", err)
	}
	c.lastChartID = msg.MessageID

	return nil
}

func (c telegramNotifyService) UpdateLastChart(chart Chart, text string) error {
	if c.lastChartID == 0 {
		return nil
	}

	cnf := tgbotapi.NewPhoto(c.telegramChat, tgbotapi.FilePath(chart.Path))
	if text != "" {
		cnf.Caption = text
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
