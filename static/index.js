/*
 * Copyright (C) 2012 Brian Ferris <bdferris@onebusaway.org>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *         http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

function Init() {
  
  var hostandport = window.location.hostname + ':' + window.location.port;

  // Create Leaflet map
  var CreateMap = function() {
    var map_canvas = document.getElementById("map_canvas");
    // Ensure container has size (index.css sets it)
    var map = L.map(map_canvas).setView([42.349, -71.059], 8);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '&copy; OpenStreetMap contributors'
    }).addTo(map);
    return map;
  };
  
  var map = CreateMap();

  // Color utilities (unchanged)
  var HsvToRgb = function(h, s, v) {
    h_int = parseInt(h * 6);
    f = h * 6 - h_int;
    var a = v * (1 - s);
    var b = v * (1 - f * s);
    var c = v * (1 - (1 - f) * s);
    switch (h_int) {
    case 0:
      return [ v, c, a ];
    case 1:
      return [ b, v, a ];
    case 2:
      return [ a, v, c ];
    case 3:
      return [ a, b, v ];
    case 4:
      return [ c, a, v ];
    case 5:
      return [ v, a, b ];
    }
  };

  var HsvToRgbString = function(h, s, v) {
    var rgb = HsvToRgb(h, s, v);
    for ( var i = 0; i < rgb.length; ++i) {
      rgb[i] = parseInt(rgb[i] * 256)
    }
    return 'rgb(' + rgb[0] + ',' + rgb[1] + ',' + rgb[2] + ')';
  };

  var h = Math.random();
  var golden_ratio_conjugate = 0.618033988749895;

  var NextRandomColor = function() {
    h = (h + golden_ratio_conjugate) % 1;
    return HsvToRgbString(h, 0.90, 0.90)
  };

  var icon = L.icon({
    iconUrl: 'http://' + hostandport + '/WhiteCircle8.png',
    iconSize: [8, 8],
    iconAnchor: [4, 4]
  });

  var CreateVehicle = function(v_data) {
    var point = L.latLng(v_data.lat, v_data.lon);
    var path = [point];
    var marker = L.marker(point, { icon: icon, title: 'id=' + v_data.id }).addTo(map);
    var polyline = L.polyline(path, {
      color: NextRandomColor(),
      opacity: 0.8,
      weight: 4
    }).addTo(map);
    return {
      id : v_data.id,
      marker : marker,
      polyline : polyline,
      path : path,
      lastUpdate : v_data.lastUpdate
    };
  };

  function CreateVehicleUpdateOperation(vehicle, lat, lon) {
    return function() {
      var point = L.latLng(lat, lon);
      vehicle.marker.setLatLng(point);
      var path = vehicle.path;
      path[path.length - 1] = point;
      vehicle.polyline.setLatLngs(path);
    };
  };
  
  var vehicles_by_id = {};
  var animation_steps = 20;

  function UpdateVehicle(v_data, updates) {
    var id = v_data.id;
    if (!(id in vehicles_by_id)) {
      vehicles_by_id[id] = CreateVehicle(v_data);
    }
    var vehicle = vehicles_by_id[id];
    if (vehicle.lastUpdate >= v_data.lastUpdate) {
      return;
    }
    vehicle.lastUpdate = v_data.lastUpdate

    var path = vehicle.path;
    var last = path[path.length - 1];
    path.push(last);

    var lat_delta = (v_data.lat - last.lat) / animation_steps;
    var lon_delta = (v_data.lon - last.lng) / animation_steps;

    if (lat_delta != 0 && lon_delta != 0) {
      for ( var i = 0; i < animation_steps; ++i) {
        var lat = last.lat + lat_delta * (i + 1);
        var lon = last.lng + lon_delta * (i + 1);
        var op = CreateVehicleUpdateOperation(vehicle, lat, lon);
        updates[i].push(op);
      }
    }
  };
  
  var first_update = true;
  
  var ProcessVehicleData = function(data) {
    var vehicles = jQuery.parseJSON(data);
    var updates = [];
    var bounds = L.latLngBounds();
    for ( var i = 0; i < animation_steps; ++i) {
      updates.push(new Array());
    }
    jQuery.each(vehicles, function() {
      UpdateVehicle(this, updates);
      bounds.extend(L.latLng(this.lat, this.lon));
    });
    if (first_update && bounds.isValid()) {
      map.fitBounds(bounds);
      first_update = false;
    }
    var applyUpdates = function() {
      if (updates.length == 0) {
        return;
      }
      var fs = updates.shift();
      for ( var i = 0; i < fs.length; i++) {
        fs[i]();
      }
      setTimeout(applyUpdates, 1);
    };
    setTimeout(applyUpdates, 1); 
  };

  // WebSocket listener
  if ("WebSocket" in window) {
    var ws = new WebSocket("ws://" + hostandport + "/data.json");
    ws.onopen = function() {
      console.log("WebSockets connection opened");
    }
    ws.onmessage = function(e) {
      console.log("Got WebSockets message");
      ProcessVehicleData(e.data);
    }
    ws.onclose = function() {
      console.log("WebSockets connection closed");
    }
  } else {
    alert("No WebSockets support");
  }
}