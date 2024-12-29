# Wind Alert BTA

A service for monitoring wind speed based on [BTA](https://ru.wikipedia.org/wiki/%D0%91%D0%A2%D0%90_(%D1%82%D0%B5%D0%BB%D0%B5%D1%81%D0%BA%D0%BE%D0%BF)) data. Sends alerts when a specified wind speed threshold is exceeded and publishes weather map updates to [the Telegram chat](https://t.me/WeatherAlertBTA).

# Features

- Checks weather data from a specified URL.
- Sends messages when wind speed exceeds the set threshold.
- Notifies when wind speed drops below the threshold after a defined interval.
- Publishes a weather chart with alerts and updates.
- Operates autonomously with configuration via environment variables.

# Installation and Configuration

```shell
 git clone git@github.com:GetSky/WeatherAlertBTA.git
 cd WeatherAlertBTA
 # Ensure you have Go installed (version 1.20+).
 go mod tidy
```

Create a .env file in the root directory and add the following variables:
```
BOT_TOKEN=<Your Telegram Bot Token>
TELEGRAM_CHAT_ID=<Chat ID to send alerts>
WEATHER_URL=https://www.sao.ru/tb/tcs/meteo/data/meteo-today.dat
CHART_WEATHER_URL=https://www.sao.ru/tb/tcs/meteo/data/meteo_today.png
WIND_THRESHOLD=15.0
POLL_INTERVAL=1m
DELAY_TIME_IN_MINUTES=20m
RESERVE_TIME_BEFORE_DUSK_IN_MINUTES=120m
```
Run the application:

```shell
go run main.go
```

Or using Docker:

```shell
docker build -t telegram-wind-alert-bta .
docker run --env-file .env --restart=always telegram-wind-alert-bta
```


# Contribution and Support

If you find a bug or have an idea for improvement, feel free to open an issue or submit a pull request. We appreciate your contributions!
