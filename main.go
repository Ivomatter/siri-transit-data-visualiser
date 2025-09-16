package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	httpPort        = flag.Int("port", 8080, "HTTP port")
	shutdownTimeout = flag.Duration("shutdown_timeout", 10*time.Second, "HTTP server shutdown timeout")
	gtfsrtURL       = flag.String("gtfsrt_url", "", "GTFS-RT vehicle positions URL (protobuf)")
	siriXmlURL      = flag.String("siri_xml_url", "", "SIRI VehicleMonitoring XML URL")
	siriJsonURL     = flag.String("siri_json_url", "", "SIRI VehicleMonitoring JSON URL")
	refreshMinSecs  = flag.Int("refresh_min_secs", 10, "Minimum refresh interval in seconds")
)

func main() {
	flag.Parse()

	feed := selectFeed()
	poll := newPoller(feed, *refreshMinSecs)
	globalPoller = poll

	mux := http.NewServeMux()
	registerRoutes(mux)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", *httpPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server starting on http://localhost:%d/", *httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// start poller
	pctx, pcancel := context.WithCancel(context.Background())
	go poll.run(pctx)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Printf("shutdown initiated...")

	pcancel()

	ctx, cancel := context.WithTimeout(context.Background(), *shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Printf("HTTP server shut down successfully")
	}
}

func selectFeed() VehicleFeedSource {
	count := 0
	if *gtfsrtURL != "" {
		count++
	}
	if *siriXmlURL != "" {
		count++
	}
	if *siriJsonURL != "" {
		count++
	}
	if count != 1 {
		log.Fatalf("provide exactly one of --gtfsrt_url, --siri_xml_url, --siri_json_url")
	}
	if *gtfsrtURL != "" {
		return NewGtfsRtVehicleFeedSource(*gtfsrtURL, 10*time.Second)
	}
	if *siriXmlURL != "" {
		return NewSiriXmlVehicleFeedSource(*siriXmlURL, 10*time.Second)
	}
	return NewSiriJsonVehicleFeedSource(*siriJsonURL, 10*time.Second)
}
