Based on [onebusaway-gtfs-realtime-visualizer](https://github.com/OneBusAway/onebusaway-gtfs-realtime-visualizer)

A Go reimplementation that serves the same frontend contract (`/data.json` streaming `{id,lat,lon,lastUpdate}`) and static UI, with adapters for GTFS-RT and SIRI (XML/JSON).

Quick start
- GTFS-RT (VehiclePositions):
  - `go run . --gtfsrt_url="" -port=8090`
- SIRI JSON (VehicleMonitoring):
  - `go run . --siri_json_url="" -port=8090`
- SIRI XML (VehicleMontiroing):
  - `go run . --siri_xml_url="https://api.entur.io/realtime/v1/rest/vm?datasetId=VYX&useOriginalId=true" --refresh_min_secs=20 -port=8090`

Open the UI at `http://localhost:<port>/`.

Google maps has been repleaced with OpenStreetMaps (Leaflet) to remove watermark 