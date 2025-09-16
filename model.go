package main

// Vehicle is the normalized model expected by the frontend.
type Vehicle struct {
	ID         string  `json:"id"`
	Lat        float64 `json:"lat"`
	Lon        float64 `json:"lon"`
	LastUpdate int64   `json:"lastUpdate"`
}
