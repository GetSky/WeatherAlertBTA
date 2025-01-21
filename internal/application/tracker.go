// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package application

import "fmt"

type State interface {
	check() error
	SetTracker(tracker *WeatherTracker)
}

type WeatherTracker struct {
	turnedOn  State
	turnedOff State

	currentState State
}

func NewWeatherTracker(turnedOn State, turnedOff State) *WeatherTracker {
	t := &WeatherTracker{
		turnedOn:  turnedOn,
		turnedOff: turnedOff,
	}
	turnedOn.SetTracker(t)
	turnedOff.SetTracker(t)
	t.switchState(turnedOff)
	return t
}

func (t *WeatherTracker) switchState(s State) {
	t.currentState = s
}

func (t *WeatherTracker) Check() {
	err := t.currentState.check()
	if err != nil {
		fmt.Printf("Tracker → %v\n", err)
	}
}
