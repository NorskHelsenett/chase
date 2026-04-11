package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
)

// PingEvent represents a single ping result for SSE streaming
type PingEvent struct {
	ServerID       uint      `json:"server_id"`
	StatusCode     int       `json:"status_code"`
	ExpectedStatus int       `json:"expected_status"`
	ResponseTime   float64   `json:"response_time_ms"`
	Error          string    `json:"error,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// DaySummary represents one day of aggregated ping results
type DaySummary struct {
	Date       string  `json:"date"`
	Total      int     `json:"total"`
	Successful int     `json:"successful"`
	Uptime     float64 `json:"uptime"`
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
func BroadcastPing(serverID uint, expectedStatus int, result PingResult) {
	evt := PingEvent{
		ServerID:       serverID,
		StatusCode:     result.StatusCode,
		ExpectedStatus: expectedStatus,
		ResponseTime:   result.ResponseTime,
		Error:          result.Error,
		Timestamp:      result.Timestamp,
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

// serverInitData is the initial payload sent per server on SSE connect
type serverInitData struct {
	ServerID       uint         `json:"server_id"`
	ExpectedStatus int          `json:"expected_status"`
	Latest         *PingEvent   `json:"latest"`
	Days           []DaySummary `json:"days"`
}

// PingStreamSSE handles GET /api/servers/pings/stream
// Sends daily-aggregated ping data plus latest ping per server, then streams live updates.
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

	db := database.GetDB()
	var servers []Server
	if err := db.Where("active = ?", true).Find(&servers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch servers"})
		return
	}

	cutoff := time.Now().AddDate(0, 0, -14) // Last 14 days
	weekAgo := time.Now().AddDate(0, 0, -7)

	for _, srv := range servers {
		type dayRow struct {
			Day        string
			Total      int
			Successful int
		}

		// Raw pings (last ~7 days)
		var rawRows []dayRow
		if err := db.Model(&PingResult{}).
			Select("DATE(timestamp) as day, COUNT(*) as total, SUM(CASE WHEN status_code = ? AND error = '' THEN 1 ELSE 0 END) as successful", srv.ExpectedStatusCode).
			Where("server_id = ? AND timestamp >= ?", srv.ID, cutoff).
			Group("DATE(timestamp)").
			Order("day ASC").
			Scan(&rawRows).Error; err != nil {
			rawRows = nil
		}

		// Hourly summaries (7-14 days ago, already aggregated)
		var hourlyRows []dayRow
		if err := db.Model(&PingHourlySummary{}).
			Select("DATE(hour) as day, SUM(total) as total, SUM(successful) as successful").
			Where("server_id = ? AND hour >= ? AND hour < ?", srv.ID, cutoff, weekAgo).
			Group("DATE(hour)").
			Order("day ASC").
			Scan(&hourlyRows).Error; err != nil {
			hourlyRows = nil
		}

		// Merge: hourly rows first, then raw rows (raw wins on overlap)
		dayMap := make(map[string]dayRow)
		for _, r := range hourlyRows {
			dayMap[r.Day] = r
		}
		for _, r := range rawRows {
			if existing, ok := dayMap[r.Day]; ok {
				r.Total += existing.Total
				r.Successful += existing.Successful
			}
			dayMap[r.Day] = r
		}

		// Sort by date
		sortedKeys := make([]string, 0, len(dayMap))
		for k := range dayMap {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		days := make([]DaySummary, len(sortedKeys))
		for i, k := range sortedKeys {
			r := dayMap[k]
			uptime := 0.0
			if r.Total > 0 {
				uptime = float64(r.Successful) / float64(r.Total) * 100
			}
			days[i] = DaySummary{
				Date:       r.Day,
				Total:      r.Total,
				Successful: r.Successful,
				Uptime:     uptime,
			}
		}

		// Get latest ping for current status
		var latest PingResult
		var latestEvent *PingEvent
		if err := db.Model(&PingResult{}).
			Select("server_id, status_code, response_time, error, timestamp").
			Where("server_id = ?", srv.ID).
			Order("timestamp DESC").
			First(&latest).Error; err == nil {
			status := latest.StatusCode
			if latest.Error != "" {
				status = 0
			}
			latestEvent = &PingEvent{
				ServerID:     latest.ServerID,
				StatusCode:   status,
				ResponseTime: latest.ResponseTime,
				Error:        latest.Error,
				Timestamp:    latest.Timestamp,
			}
		}

		initData := serverInitData{
			ServerID:       srv.ID,
			ExpectedStatus: srv.ExpectedStatusCode,
			Latest:         latestEvent,
			Days:           days,
		}

		data, err := json.Marshal(initData)
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
