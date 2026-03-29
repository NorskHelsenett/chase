package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
)

// PingEvent represents a ping result for SSE streaming
type PingEvent struct {
	ServerID     uint      `json:"server_id"`
	StatusCode   int       `json:"status_code"`
	ResponseTime float64   `json:"response_time_ms"`
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// pingHub manages SSE clients for ping result streaming
type pingHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

var hub = &pingHub{
	clients: make(map[chan []byte]struct{}),
}

func (h *pingHub) subscribe() chan []byte {
	ch := make(chan []byte, 64)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *pingHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	close(ch)
	h.mu.Unlock()
}

// BroadcastPing sends a ping result to all connected SSE clients
func BroadcastPing(serverID uint, result PingResult) {
	evt := PingEvent{
		ServerID:     serverID,
		StatusCode:   result.StatusCode,
		ResponseTime: result.ResponseTime,
		Error:        result.Error,
		Timestamp:    result.Timestamp,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: ping\ndata: %s\n\n", data))

	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for ch := range hub.clients {
		select {
		case ch <- msg:
		default:
			// client too slow, drop message
		}
	}
}

// PingStreamSSE handles GET /api/servers/pings/stream
// Sends initial ping data for all servers, then streams updates in real-time.
func PingStreamSSE(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Send initial ping data for all active servers
	db := database.GetDB()
	var servers []Server
	if err := db.Where("active = ?", true).Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch servers"})
		return
	}

	type serverPings struct {
		ServerID uint        `json:"server_id"`
		Pings    []PingEvent `json:"pings"`
	}

	for _, srv := range servers {
		var pings []PingResult
		if err := db.Model(&PingResult{}).
			Select("server_id, status_code, response_time, error, timestamp").
			Where("server_id = ?", srv.ID).
			Order("timestamp DESC").
			Limit(10).
			Find(&pings).Error; err != nil {
			continue
		}

		events := make([]PingEvent, len(pings))
		for i, p := range pings {
			status := p.StatusCode
			if p.Error != "" {
				status = 0
			}
			events[i] = PingEvent{
				ServerID:     p.ServerID,
				StatusCode:   status,
				ResponseTime: p.ResponseTime,
				Error:        p.Error,
				Timestamp:    p.Timestamp,
			}
		}

		sp := serverPings{ServerID: srv.ID, Pings: events}
		data, err := json.Marshal(sp)
		if err != nil {
			continue
		}
		fmt.Fprintf(c.Writer, "event: init\ndata: %s\n\n", data)
	}
	flusher.Flush()

	// Subscribe to real-time updates
	ch := hub.subscribe()
	defer hub.unsubscribe(ch)

	ctx := c.Request.Context()
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := c.Writer.Write(msg); err != nil {
				log.Printf("SSE write error: %v", err)
				return
			}
			flusher.Flush()
		case <-keepAlive.C:
			if _, err := fmt.Fprintf(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
