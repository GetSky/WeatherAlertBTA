// Copyright 2024 Alexander Getmansky <alex@getsky.tech>
// Licensed under the Apache License, Version 2.0

package application

import (
	"time"
)

type TurnOffState struct {
	tracker     *WeatherTracker
	scheduleSrv ScheduleService
	notifySrv   NotifyService
}

func NewTurnOffState(schedule ScheduleService, notify NotifyService) *TurnOffState {
	return &TurnOffState{
		scheduleSrv: schedule,
		notifySrv:   notify,
	}
}

func (t *TurnOffState) SetTracker(tracker *WeatherTracker) {
	t.tracker = tracker
}

func (t *TurnOffState) check() error {
	isWorkTime, err := t.scheduleSrv.IsWorkNow()
	if err != nil {
		return err
	}

	if isWorkTime == false {
		return nil
	}

	dusk, dawn, err := t.scheduleSrv.GetNautical(time.Now())
	if err != nil {
		return err
	}

	err = t.notifySrv.SendWorkStarted(dusk, dawn)
	if err != nil {
		return err
	}

	t.tracker.switchState(t.tracker.turnedOn)
	t.tracker.Check()

	return nil
}
