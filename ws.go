package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

var hub = &wsHub{clients: make(map[*websocket.Conn]struct{})}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	hub.add(conn)
	go readPump(conn)

	// Send the most recent snapshot if available to center the map quickly
	if snapshot := getLastSnapshot(); len(snapshot) > 0 {
		_ = writeVehicles(conn, snapshot)
	} else {
		_ = writeVehicles(conn, nil)
	}
}

func (h *wsHub) add(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *wsHub) remove(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *wsHub) broadcast(vehicles []Vehicle) {
	data, _ := json.Marshal(vehicles)
	h.mu.Lock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			c.Close()
			delete(h.clients, c)
		}
	}
	h.mu.Unlock()
}

func readPump(c *websocket.Conn) {
	defer func() {
		hub.remove(c)
		_ = c.Close()
	}()
	for {
		if _, _, err := c.ReadMessage(); err != nil {
			return
		}
	}
}

func writeVehicles(c *websocket.Conn, v []Vehicle) error {
	data, _ := json.Marshal(v)
	return c.WriteMessage(websocket.TextMessage, data)
}
