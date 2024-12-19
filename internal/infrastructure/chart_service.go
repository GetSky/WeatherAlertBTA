package infrastructure

import (
	"fmt"
	. "github.com/GetSky/WeatherAlertBTA/internal/application"
	"io"
	"net/http"
	"os"
	"time"
)

const tmpFileName = "chart.png"

type chartService struct {
	chartURL string
	tmpPath  string
}

func NewChartService(
	chartWeatherURL string,
) ChartService {
	return &chartService{
		chartURL: chartWeatherURL,
	}
}

func (c chartService) GetUpdatedChart() (Chart, error) {
	chart, err := c.downloadChart()
	if err != nil {
		return Chart{}, fmt.Errorf("downloadChart → %s", err)
	}
	return chart, nil
}

// downloadChart
// Upload the image to ourselves, because if send it to a direct URI, Telegram will not allow us to reload it later.
func (c chartService) downloadChart() (Chart, error) {
	resp, err := http.Get(c.chartURL)
	if err != nil {
		return Chart{}, fmt.Errorf("failed to download image: %v", err)
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return Chart{}, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	file, err := os.Create(tmpFileName)
	if err != nil {
		return Chart{}, fmt.Errorf("failed to create file: %v", err)
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return Chart{}, fmt.Errorf("failed to save image: %v", err)
	}

	return Chart{
		Path:     tmpFileName,
		CreateAt: time.Now(),
	}, nil
}
