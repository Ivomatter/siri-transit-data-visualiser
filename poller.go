package main

import (
	"context"
	"log"
	"sync"
	"time"
)

// poller maintains periodic fetches and broadcasts updates when vehicles change.

type poller struct {
	feed              VehicleFeedSource
	minRefreshSeconds int
	mu                sync.Mutex
	lastVehicles      map[string]Vehicle
	mostRecentFetchMs int64
}

func newPoller(feed VehicleFeedSource, minRefreshSeconds int) *poller {
	return &poller{
		feed:              feed,
		minRefreshSeconds: minRefreshSeconds,
		lastVehicles:      make(map[string]Vehicle),
	}
}

func (p *poller) run(ctx context.Context) {
	interval := time.Duration(p.minRefreshSeconds) * time.Second
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			start := time.Now()
			p.tick(ctx)
			elapsed := time.Since(start)
			if p.mostRecentFetchMs != 0 {
				interval = maxDuration(elapsed/2, time.Duration(p.minRefreshSeconds)*time.Second)
			}
			t.Reset(interval)
		}
	}
}

func (p *poller) tick(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	vehicles, err := p.feed.Fetch(cctx)
	if err != nil {
		log.Printf("poll error: %v", err)
		return
	}
	log.Printf("fetched vehicles: %d", len(vehicles))
	p.mostRecentFetchMs = time.Now().UnixMilli()

	changed, snapshot := p.detectChanges(vehicles)
	if changed {
		log.Printf("vehicles updated: %d", len(snapshot))
		hub.broadcast(snapshot)
	}
}

func (p *poller) detectChanges(in []Vehicle) (bool, []Vehicle) {
	p.mu.Lock()
	defer p.mu.Unlock()
	changed := false
	current := make(map[string]Vehicle, len(in))
	for _, v := range in {
		prev, ok := p.lastVehicles[v.ID]
		if !ok || prev.Lat != v.Lat || prev.Lon != v.Lon {
			v.LastUpdate = time.Now().UnixMilli()
			changed = true
		} else {
			v.LastUpdate = prev.LastUpdate
		}
		current[v.ID] = v
	}
	p.lastVehicles = current
	// build stable slice for broadcast
	out := make([]Vehicle, 0, len(current))
	for _, v := range current {
		out = append(out, v)
	}
	return changed, out
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

// getLastSnapshot returns a best-effort copy of the last vehicle list.
func getLastSnapshot() []Vehicle {
	// Access poller's lastVehicles would require a reference; for now, we keep a global pointer.
	if globalPoller == nil {
		return nil
	}
	globalPoller.mu.Lock()
	defer globalPoller.mu.Unlock()
	out := make([]Vehicle, 0, len(globalPoller.lastVehicles))
	for _, v := range globalPoller.lastVehicles {
		out = append(out, v)
	}
	return out
}

var globalPoller *poller
