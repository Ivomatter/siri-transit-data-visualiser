package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type SiriJsonVehicleFeedSource struct {
	url        string
	httpClient *http.Client
}

func NewSiriJsonVehicleFeedSource(url string, timeout time.Duration) *SiriJsonVehicleFeedSource {
	return &SiriJsonVehicleFeedSource{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *SiriJsonVehicleFeedSource) Fetch(ctx context.Context) ([]Vehicle, error) {
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
		return nil, fmt.Errorf("siri json http status: %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Minimal schema-walking: Siri?.ServiceDelivery.VehicleMonitoringDelivery[].VehicleActivity[]
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, err
	}
	// Handle optional top-level "Siri" wrapper
	if siri, ok := root["Siri"].(map[string]any); ok && siri != nil {
		root = siri
	}
	sd, _ := root["ServiceDelivery"].(map[string]any)
	vmdArr, _ := sd["VehicleMonitoringDelivery"].([]any)
	vehicles := make([]Vehicle, 0, 256)
	for _, vmdAny := range vmdArr {
		vmd, _ := vmdAny.(map[string]any)
		vaArr, _ := vmd["VehicleActivity"].([]any)
		for _, vaAny := range vaArr {
			va, _ := vaAny.(map[string]any)
			mvj, _ := va["MonitoredVehicleJourney"].(map[string]any)
			if mvj == nil {
				continue
			}
			id := stringFrom(mvj["VehicleRef"])
			if id == "" {
				id = stringFromNested(mvj, "FramedVehicleJourneyRef", "DatedVehicleJourneyRef")
			}
			lat, lon := floatFromNested(mvj, "VehicleLocation", "Latitude"), floatFromNested(mvj, "VehicleLocation", "Longitude")
			if id == "" || (lat == 0 && lon == 0) {
				continue
			}
			vehicles = append(vehicles, Vehicle{ID: id, Lat: lat, Lon: lon})
		}
	}
	return vehicles, nil
}

func stringFrom(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringFromNested(m map[string]any, k1, k2 string) string {
	m1, _ := m[k1].(map[string]any)
	return stringFrom(m1[k2])
}

func floatFromNested(m map[string]any, k1, k2 string) float64 {
	m1, _ := m[k1].(map[string]any)
	switch v := m1[k2].(type) {
	case float64:
		return v
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}
