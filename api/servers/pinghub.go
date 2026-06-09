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
	ServerID        uint      `json:"server_id"`
	StatusCode      int       `json:"status_code"`
	ExpectedStatus  int       `json:"expected_status"`
	ResponseTime    float64   `json:"response_time_ms"`
	Error           string    `json:"error,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
	Favicon         string    `json:"favicon,omitempty"`
	SiteTitle       string    `json:"site_title,omitempty"`
	SiteDescription string    `json:"site_description,omitempty"`
	OGImage         string    `json:"og_image,omitempty"`
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
		ServerID:        serverID,
		StatusCode:      result.StatusCode,
		ExpectedStatus:  expectedStatus,
		ResponseTime:    result.ResponseTime,
		Error:           result.Error,
		Timestamp:       result.Timestamp,
		Favicon:         result.siteMetadata.Favicon,
		SiteTitle:       result.siteMetadata.Title,
		SiteDescription: result.siteMetadata.Description,
		OGImage:         result.siteMetadata.OGImage,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	msg := []byte(fmt.Sprintf("event: ping\ndata: %s\n\n", data))

	broadcast(msg)
}

// broadcast fans a pre-formatted SSE message out to every connected client,
// dropping it for any client whose buffer is full rather than blocking.
func broadcast(msg []byte) {
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

// BroadcastServerAdded notifies all SSE clients that a new server now exists so
// grids and lists can add it live, without a manual refresh or page reload.
func BroadcastServerAdded(server Server) {
	data, err := json.Marshal(server)
	if err != nil {
		return
	}
	broadcast([]byte(fmt.Sprintf("event: server_added\ndata: %s\n\n", data)))
}

// BroadcastServersChanged is a lightweight signal that the server set changed in
// bulk (e.g. a batch import) and clients should refetch. Used instead of a flood
// of per-server events, which a slow client's buffer would drop.
func BroadcastServersChanged() {
	broadcast([]byte("event: servers_changed\ndata: {}\n\n"))
}

// serverInitData is the initial payload sent per server on SSE connect
type serverInitData struct {
	ServerID        uint         `json:"server_id"`
	ExpectedStatus  int          `json:"expected_status"`
	Latest          *PingEvent   `json:"latest"`
	Days            []DaySummary `json:"days"`
	Favicon         string       `json:"favicon,omitempty"`
	SiteTitle       string       `json:"site_title,omitempty"`
	SiteDescription string       `json:"site_description,omitempty"`
	OGImage         string       `json:"og_image,omitempty"`
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

	// Fetch everything the init payload needs in a few set-based queries rather
	// than ~3 per server, which melts the DB once there are thousands of servers.

	// Latest ping per active server.
	type latestRow struct {
		ServerID     uint
		StatusCode   int
		ResponseTime float64
		Error        string
		Timestamp    time.Time
	}
	latestByServer := make(map[uint]latestRow)
	{
		var rows []latestRow
		db.Raw(`SELECT DISTINCT ON (pr.server_id)
		            pr.server_id, pr.status_code, pr.response_time, pr.error, pr.timestamp
		        FROM ping_results pr
		        JOIN servers s ON s.id = pr.server_id AND s.active = true
		        ORDER BY pr.server_id, pr.timestamp DESC`).Scan(&rows)
		for _, r := range rows {
			latestByServer[r.ServerID] = r
		}
	}

	// Daily uptime, last 14 days, for every active server at once. Hourly
	// summaries cover the older week; raw pings cover the rest. They're summed
	// per (server, day) — matching the original additive per-server merge.
	type dayRow struct {
		ServerID   uint
		Day        string
		Total      int
		Successful int
	}
	daysByServer := make(map[uint]map[string]DaySummary)
	addDay := func(serverID uint, day string, total, successful int) {
		m := daysByServer[serverID]
		if m == nil {
			m = make(map[string]DaySummary)
			daysByServer[serverID] = m
		}
		existing := m[day]
		total += existing.Total
		successful += existing.Successful
		uptime := 0.0
		if total > 0 {
			uptime = float64(successful) / float64(total) * 100
		}
		m[day] = DaySummary{Date: day, Total: total, Successful: successful, Uptime: uptime}
	}

	{
		var rows []dayRow
		db.Raw(`SELECT server_id, to_char(hour, 'YYYY-MM-DD') as day,
		            SUM(total) as total, SUM(successful) as successful
		        FROM ping_hourly_summaries
		        WHERE hour >= ? AND hour < ?
		            AND server_id IN (SELECT id FROM servers WHERE active = true)
		        GROUP BY server_id, to_char(hour, 'YYYY-MM-DD')`, cutoff, weekAgo).Scan(&rows)
		for _, r := range rows {
			addDay(r.ServerID, r.Day, r.Total, r.Successful)
		}
	}
	{
		var rows []dayRow
		db.Raw(`SELECT pr.server_id, to_char(pr.timestamp, 'YYYY-MM-DD') as day,
		            COUNT(*) as total,
		            SUM(CASE WHEN pr.status_code = s.expected_status_code AND pr.error = '' THEN 1 ELSE 0 END) as successful
		        FROM ping_results pr
		        JOIN servers s ON s.id = pr.server_id AND s.active = true
		        WHERE pr.timestamp >= ?
		        GROUP BY pr.server_id, to_char(pr.timestamp, 'YYYY-MM-DD')`, cutoff).Scan(&rows)
		for _, r := range rows {
			addDay(r.ServerID, r.Day, r.Total, r.Successful)
		}
	}

	// Emit one init event per server, flushing in batches so the client renders
	// progressively instead of waiting for the whole fleet to be marshaled.
	const flushEvery = 50
	for i := range servers {
		srv := servers[i]

		dayMap := daysByServer[srv.ID]
		sortedKeys := make([]string, 0, len(dayMap))
		for k := range dayMap {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)
		days := make([]DaySummary, len(sortedKeys))
		for j, k := range sortedKeys {
			days[j] = dayMap[k]
		}

		var latestEvent *PingEvent
		if lr, ok := latestByServer[srv.ID]; ok {
			status := lr.StatusCode
			if lr.Error != "" {
				status = 0
			}
			latestEvent = &PingEvent{
				ServerID:     lr.ServerID,
				StatusCode:   status,
				ResponseTime: lr.ResponseTime,
				Error:        lr.Error,
				Timestamp:    lr.Timestamp,
			}
		}

		initData := serverInitData{
			ServerID:        srv.ID,
			ExpectedStatus:  srv.ExpectedStatusCode,
			Latest:          latestEvent,
			Days:            days,
			Favicon:         srv.Favicon,
			SiteTitle:       srv.SiteTitle,
			SiteDescription: srv.SiteDescription,
			OGImage:         srv.OGImage,
		}

		data, err := json.Marshal(initData)
		if err != nil {
			continue
		}
		if _, err := fmt.Fprintf(c.Writer, "event: init\ndata: %s\n\n", data); err != nil {
			return
		}
		if (i+1)%flushEvery == 0 {
			flusher.Flush()
		}
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
