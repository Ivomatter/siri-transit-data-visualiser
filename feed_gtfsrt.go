package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	"google.golang.org/protobuf/proto"
)

type VehicleFeedSource interface {
	Fetch(ctx context.Context) ([]Vehicle, error)
}

type GtfsRtVehicleFeedSource struct {
	url        string
	httpClient *http.Client
}

func NewGtfsRtVehicleFeedSource(url string, timeout time.Duration) *GtfsRtVehicleFeedSource {
	return &GtfsRtVehicleFeedSource{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *GtfsRtVehicleFeedSource) Fetch(ctx context.Context) ([]Vehicle, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gtfs-rt http status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var feed gtfs.FeedMessage
	if err := proto.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	vehicles := make([]Vehicle, 0, len(feed.Entity))
	for _, ent := range feed.Entity {
		if ent == nil || ent.Vehicle == nil {
			continue
		}
		vp := ent.Vehicle
		if vp.Vehicle == nil || vp.Position == nil {
			continue
		}
		id := vp.Vehicle.Id
		if id == nil || *id == "" {
			continue
		}
		lat := vp.Position.Latitude
		lon := vp.Position.Longitude
		if lat == nil || lon == nil {
			continue
		}
		v := Vehicle{
			ID:  *id,
			Lat: float64(*lat),
			Lon: float64(*lon),
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, nil
}
