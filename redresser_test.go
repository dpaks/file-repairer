package main

import (
	"context"
	"testing"
	"time"
)

func Test_consumeFileEvents(t *testing.T) {
	var (
		ctx    context.Context
		feeder chan Event
		err    chan error
	)

	ctx, cancel := context.WithCancel(context.Background())
	feeder = make(chan Event, 1)
	err = make(chan error)

	name := "sanity"
	ticker := time.NewTicker(time.Second)
	select {
	case <-ticker.C:
		cancel()
		ticker.Stop()
	default:
		t.Run(name, func(t *testing.T) {
			go consumeFileEvents(ctx, feeder, err)
		})
	}

}
