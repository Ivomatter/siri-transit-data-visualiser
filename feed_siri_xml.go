package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type SiriXmlVehicleFeedSource struct {
	url        string
	httpClient *http.Client
}

func NewSiriXmlVehicleFeedSource(url string, timeout time.Duration) *SiriXmlVehicleFeedSource {
	return &SiriXmlVehicleFeedSource{
		url:        url,
		httpClient: &http.Client{Timeout: timeout},
	}
}

// Minimal streaming extraction for SIRI VM XML (namespace tolerant via Name.Local)
func (s *SiriXmlVehicleFeedSource) Fetch(ctx context.Context) ([]Vehicle, error) {
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
		return nil, fmt.Errorf("siri xml http status: %d", resp.StatusCode)
	}
	dec := xml.NewDecoder(resp.Body)

	var (
		inSiri, inSD, inVMD, inVA, inMVJ, inVL bool
		curID                                  string
		curLat, curLon                         string
		vehicles                               []Vehicle
	)

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch se := tok.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "Siri":
				inSiri = true
			case "ServiceDelivery":
				if inSiri {
					inSD = true
				}
			case "VehicleMonitoringDelivery":
				if inSD {
					inVMD = true
				}
			case "VehicleActivity":
				if inVMD {
					inVA = true
					curID, curLat, curLon = "", "", ""
				}
			case "MonitoredVehicleJourney":
				if inVA {
					inMVJ = true
				}
			case "VehicleLocation":
				if inMVJ || inVA {
					inVL = true
				}
			case "VehicleRef":
				if inMVJ || inVA {
					var v string
					if err := dec.DecodeElement(&v, &se); err == nil {
						curID = v
					}
				}
			case "Latitude":
				if inVL {
					var v string
					if err := dec.DecodeElement(&v, &se); err == nil {
						curLat = v
					}
				}
			case "Longitude":
				if inVL {
					var v string
					if err := dec.DecodeElement(&v, &se); err == nil {
						curLon = v
					}
				}
			}
		case xml.EndElement:
			switch se.Name.Local {
			case "VehicleLocation":
				inVL = false
			case "MonitoredVehicleJourney":
				inMVJ = false
			case "VehicleActivity":
				if inVA {
					inVA = false
					if curID != "" && curLat != "" && curLon != "" {
						if latf, lonf, ok := parseLatLon(curLat, curLon); ok {
							vehicles = append(vehicles, Vehicle{ID: curID, Lat: latf, Lon: lonf})
						}
					}
				}
			case "VehicleMonitoringDelivery":
				inVMD = false
			case "ServiceDelivery":
				inSD = false
			case "Siri":
				inSiri = false
			}
		}
	}
	return vehicles, nil
}

func parseLatLon(lat, lon string) (float64, float64, bool) {
	lf, err1 := strconv.ParseFloat(lat, 64)
	if err1 != nil {
		return 0, 0, false
	}
	lo, err2 := strconv.ParseFloat(lon, 64)
	if err2 != nil {
		return 0, 0, false
	}
	return lf, lo, true
}
