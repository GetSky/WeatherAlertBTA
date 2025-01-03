// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package infrastructure

import (
	"bufio"
	"fmt"
	"github.com/GetSky/WeatherAlertBTA/internal/application"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type weatherService struct {
	http         http.Client
	WeatherUrl   string
	lastModified string
	current      application.Weather
}

func NewWeatherService(url string) application.WeatherService {
	return &weatherService{
		WeatherUrl: url,
	}
}

func (w *weatherService) GetLatestWeather() (weather application.Weather, err error) {
	// Checking for an updated file
	modifiedAt, err := w.getLastUpdate(w.WeatherUrl)
	if err != nil {
		return weather, fmt.Errorf("Failed to check last update: %v\n", err)
	}

	if modifiedAt == w.lastModified {
		return w.current, nil
	}
	w.lastModified = modifiedAt

	req, err := http.NewRequest("GET", w.WeatherUrl, nil)
	if err != nil {

		return weather, fmt.Errorf("Failed to create GET request: %v\n", err)
	}
	req.Header.Set("Range", "bytes=-66")

	resp, err := w.http.Do(req)
	if err != nil {
		return weather, fmt.Errorf("Failed to fetch weather data: %v\n", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusPartialContent {
		return weather, fmt.Errorf("Unexpected HTTP response: %s\n", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Scan()
	var lastLine = scanner.Text()

	if lastLine == "" {
		return weather, fmt.Errorf("no data available")
	}

	fields := strings.Fields(lastLine)
	if len(fields) < 9 {
		return weather, fmt.Errorf("malformed last line")
	}

	windSpeed, err := strconv.ParseFloat(fields[7], 64)
	if err != nil {
		return weather, fmt.Errorf("Error parsing wind speed: %v\n", err)
	}

	temp, err := strconv.ParseFloat(fields[3], 64)
	if err != nil {
		return weather, fmt.Errorf("Error parsing temperature: %v\n", err)
	}

	timestamp := fmt.Sprintf("%s %s", fields[0], fields[1])
	updateAt, err := time.Parse("02-Jan-2006 15:04:05", timestamp)
	if err != nil {
		return weather, fmt.Errorf("Error parsing date time: %v\n", err)
	}

	w.current = application.Weather{
		UpdateAt:    updateAt,
		Temperature: temp,
		WindSpeed:   windSpeed,
	}

	return w.current, nil
}

func (w *weatherService) getLastUpdate(url string) (string, error) {
	modifiedAt := ""
	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return modifiedAt, fmt.Errorf("Failed to create HEAD request: %v\n", err)
	}

	headResp, err := w.http.Do(headReq)
	if err != nil {
		return modifiedAt, fmt.Errorf("HEAD request failed: %v\n", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(headResp.Body)

	if headResp.StatusCode != http.StatusOK {
		return modifiedAt, fmt.Errorf("Unexpected HEAD response: %s\n", headResp.Status)
	}

	modifiedAt = headResp.Header.Get("Last-Modified")
	return modifiedAt, nil
}
