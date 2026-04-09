package servers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/norskhelsenett/chase/database"
	"gorm.io/gorm"
)

type GeoResult struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Org         string  `json:"org"`
	ISP         string  `json:"isp"`
	AS          string  `json:"as"`
}

// GeoCache is the database model for persisted geo lookups
type GeoCache struct {
	gorm.Model
	IP          string  `json:"ip" gorm:"uniqueIndex"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Org         string  `json:"org"`
	ISP         string  `json:"isp"`
	AS          string  `json:"as"`
}

type ServerGeo struct {
	ServerID uint     `json:"server_id"`
	URL      string   `json:"url"`
	IPs      []string `json:"ips"`
	Status   string   `json:"status"`
}

type ServerGeoResponse struct {
	Servers  []ServerGeo          `json:"servers"`
	Geo      map[string]GeoResult `json:"geo"`
	LocalIPs []string             `json:"local_ips"` // Private IPs that can't be geo-located
}

var (
	geoMemCache   = make(map[string]*GeoResult)
	geoMemCacheMu sync.RWMutex
	geoCacheTTL   = 7 * 24 * time.Hour

	// Response-level cache for GetServersGeo, invalidated by status fingerprint or TTL
	geoResponseCache       *ServerGeoResponse
	geoResponseFingerprint string
	geoResponseCacheMu     sync.RWMutex
	geoResponseTTL         = 8 * time.Hour
	geoResponseCacheAt     time.Time
)

// InvalidateGeoResponseCache clears the geo response cache and triggers a background rebuild
// Call this when servers are added, updated, or deleted
func InvalidateGeoResponseCache() {
	geoResponseCacheMu.Lock()
	geoResponseCache = nil
	geoResponseFingerprint = ""
	geoResponseCacheMu.Unlock()

	// Trigger immediate rebuild in background
	go rebuildGeoResponseCache()
}

// privateRanges contains all RFC 1918 and other non-routable IP ranges
var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	} {
		_, network, _ := net.ParseCIDR(cidr)
		privateRanges = append(privateRanges, network)
	}
}

// isPrivateIP returns true if the IP is in a private/local/non-routable range
func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// geoProviders are tried in order until one succeeds
var geoProviders = []func(string) (*GeoResult, error){
	lookupIPWhois,
	lookupIPAPI,
}

func lookupGeo(ip string) (*GeoResult, error) {
	if isPrivateIP(ip) {
		return nil, fmt.Errorf("private IP %s skipped", ip)
	}

	// Check in-memory cache first
	geoMemCacheMu.RLock()
	if cached, ok := geoMemCache[ip]; ok {
		geoMemCacheMu.RUnlock()
		return cached, nil
	}
	geoMemCacheMu.RUnlock()

	// Check database cache
	db := database.GetDB()
	var cached GeoCache
	if err := db.Where("ip = ? AND updated_at > ?", ip, time.Now().Add(-geoCacheTTL)).First(&cached).Error; err == nil {
		result := &GeoResult{
			IP: cached.IP, Country: cached.Country, CountryCode: cached.CountryCode,
			City: cached.City, Region: cached.Region, Lat: cached.Lat, Lon: cached.Lon,
			Org: cached.Org, ISP: cached.ISP, AS: cached.AS,
		}
		geoMemCacheMu.Lock()
		geoMemCache[ip] = result
		geoMemCacheMu.Unlock()
		return result, nil
	}

	// Try custom GEO_API_URL first if configured
	if customURL := os.Getenv("GEO_API_URL"); customURL != "" {
		if result, err := lookupCustom(ip, customURL); err == nil {
			cacheGeo(ip, result)
			return result, nil
		}
	}

	for _, provider := range geoProviders {
		result, err := provider(ip)
		if err != nil {
			continue
		}
		cacheGeo(ip, result)
		return result, nil
	}

	return nil, fmt.Errorf("all geo providers failed for %s", ip)
}

func cacheGeo(ip string, result *GeoResult) {
	// Update in-memory cache
	geoMemCacheMu.Lock()
	geoMemCache[ip] = result
	geoMemCacheMu.Unlock()

	// Persist to database
	db := database.GetDB()
	entry := GeoCache{
		IP: ip, Country: result.Country, CountryCode: result.CountryCode,
		City: result.City, Region: result.Region, Lat: result.Lat, Lon: result.Lon,
		Org: result.Org, ISP: result.ISP, AS: result.AS,
	}
	var existing GeoCache
	if err := db.Where("ip = ?", ip).First(&existing).Error; err == nil {
		db.Model(&existing).Updates(entry)
	} else {
		db.Create(&entry)
	}
}

// StartGeoCacheRefresh runs background jobs:
// 1. Refreshes stale individual geo cache entries once a day
// 2. Rebuilds the geo response cache every 5 minutes for status updates
func StartGeoCacheRefresh() {
	// Individual geo entry refresh (runs daily)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once at startup
		refreshStaleGeoEntries()

		for range ticker.C {
			refreshStaleGeoEntries()
		}
	}()

	// Geo response cache rebuild (runs every 5 minutes + at startup)
	go func() {
		// Build initial cache at startup
		log.Println("Building initial geo response cache...")
		rebuildGeoResponseCache()

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rebuildGeoResponseCache()
		}
	}()
}

func refreshStaleGeoEntries() {
	db := database.GetDB()
	var stale []GeoCache
	if err := db.Where("updated_at < ?", time.Now().Add(-geoCacheTTL)).Find(&stale).Error; err != nil {
		log.Printf("Failed to fetch stale geo entries: %v", err)
		return
	}

	if len(stale) == 0 {
		return
	}

	log.Printf("Refreshing %d stale geo cache entries", len(stale))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, entry := range stale {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Clear from in-memory cache so lookupGeo hits the API
			geoMemCacheMu.Lock()
			delete(geoMemCache, ip)
			geoMemCacheMu.Unlock()

			if _, err := lookupGeo(ip); err != nil {
				log.Printf("Geo refresh failed for %s: %v", ip, err)
			}
		}(entry.IP)
	}
	wg.Wait()
	log.Printf("Geo cache refresh complete")
}

// ipwho.is — free, no key, 10k/month
func lookupIPWhois(ip string) (*GeoResult, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("https://ipwho.is/%s", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Success     bool    `json:"success"`
		Country     string  `json:"country"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region"`
		City        string  `json:"city"`
		Lat         float64 `json:"latitude"`
		Lon         float64 `json:"longitude"`
		Connection  struct {
			ISP string `json:"isp"`
			Org string `json:"org"`
			ASN int    `json:"asn"`
		} `json:"connection"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if !raw.Success {
		return nil, fmt.Errorf("ipwho.is lookup failed")
	}
	return &GeoResult{
		IP:          ip,
		Country:     raw.Country,
		CountryCode: raw.CountryCode,
		City:        raw.City,
		Region:      raw.Region,
		Lat:         raw.Lat,
		Lon:         raw.Lon,
		Org:         raw.Connection.Org,
		ISP:         raw.Connection.ISP,
		AS:          fmt.Sprintf("AS%d", raw.Connection.ASN),
	}, nil
}

// ip-api.com — free, no key, 45/min
func lookupIPAPI(ip string) (*GeoResult, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,regionName,city,lat,lon,isp,org,as", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Status      string  `json:"status"`
		Message     string  `json:"message"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"regionName"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		ISP         string  `json:"isp"`
		Org         string  `json:"org"`
		AS          string  `json:"as"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Status != "success" {
		return nil, fmt.Errorf("ip-api failed: %s", raw.Message)
	}
	return &GeoResult{
		IP:          ip,
		Country:     raw.Country,
		CountryCode: raw.CountryCode,
		City:        raw.City,
		Region:      raw.Region,
		Lat:         raw.Lat,
		Lon:         raw.Lon,
		Org:         raw.Org,
		ISP:         raw.ISP,
		AS:          raw.AS,
	}, nil
}

// Custom API compatible with ip-api.com response format
func lookupCustom(ip, baseURL string) (*GeoResult, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/%s", baseURL, ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result GeoResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	result.IP = ip
	return &result, nil
}

// computeStatusFingerprint builds a cheap hash from active server IDs and their
// latest ping statuses. This avoids the expensive DNS + geo pipeline when nothing changed.
func computeStatusFingerprint(db *gorm.DB) (string, []Server, map[uint]*PingResult) {
	var srvs []Server
	if err := db.Where("active = ?", true).Find(&srvs).Error; err != nil {
		return "", nil, nil
	}

	if len(srvs) == 0 {
		return "empty", srvs, nil
	}

	serverIDs := make([]uint, len(srvs))
	for i := range srvs {
		serverIDs[i] = srvs[i].ID
	}

	var latestPings []PingResult
	db.Where("id IN (?)",
		db.Model(&PingResult{}).
			Select("MAX(id)").
			Where("server_id IN ?", serverIDs).
			Group("server_id"),
	).Find(&latestPings)

	pingByServer := make(map[uint]*PingResult, len(latestPings))
	for i := range latestPings {
		pingByServer[latestPings[i].ServerID] = &latestPings[i]
	}

	// Build deterministic fingerprint: sorted server IDs with their status
	type entry struct {
		id     uint
		status string
	}
	entries := make([]entry, 0, len(srvs))
	for _, srv := range srvs {
		status := "unknown"
		if ping, ok := pingByServer[srv.ID]; ok {
			if ping.Error != "" {
				status = "down"
			} else if ping.StatusCode == srv.ExpectedStatusCode {
				status = "up"
			} else {
				status = "down"
			}
		}
		entries = append(entries, entry{id: srv.ID, status: status})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })

	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(h, "%d:%s,", e.id, e.status)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), srvs, pingByServer
}

// rebuildGeoResponseCache performs the expensive rebuild of the geo response cache in the background
// This includes DNS lookups and geo API calls for all servers
func rebuildGeoResponseCache() {
	db := database.GetDB()

	// Compute cheap fingerprint from server IDs + statuses (no DNS, no geo)
	fingerprint, srvs, pingByServer := computeStatusFingerprint(db)
	if srvs == nil {
		log.Println("Failed to compute status fingerprint for geo cache rebuild")
		return
	}

	if len(srvs) == 0 {
		geoResponseCacheMu.Lock()
		geoResponseCache = &ServerGeoResponse{
			Servers:  []ServerGeo{},
			Geo:      map[string]GeoResult{},
			LocalIPs: []string{},
		}
		geoResponseFingerprint = fingerprint
		geoResponseCacheAt = time.Now()
		geoResponseCacheMu.Unlock()
		return
	}

	// Check if we need to rebuild (fingerprint changed or TTL expired)
	geoResponseCacheMu.RLock()
	shouldRebuild := geoResponseCache == nil ||
		geoResponseFingerprint != fingerprint ||
		time.Since(geoResponseCacheAt) >= geoResponseTTL
	geoResponseCacheMu.RUnlock()

	if !shouldRebuild {
		return
	}

	log.Printf("Rebuilding geo response cache for %d servers...", len(srvs))
	startTime := time.Now()

	// Fetch pings with PingDetail preloaded
	serverIDs := make([]uint, len(srvs))
	for i := range srvs {
		serverIDs[i] = srvs[i].ID
	}

	var latestPings []PingResult
	db.Where("id IN (?)",
		db.Model(&PingResult{}).
			Select("MAX(id)").
			Where("server_id IN ?", serverIDs).
			Group("server_id"),
	).Preload("PingDetail").Find(&latestPings)

	pingByServerWithDetail := make(map[uint]*PingResult, len(latestPings))
	for i := range latestPings {
		pingByServerWithDetail[latestPings[i].ServerID] = &latestPings[i]
	}

	// Phase 2: Collect IPs from ping details; only DNS-resolve servers without a known IP
	type serverData struct {
		srv    *Server
		ips    []string
		status string
	}

	// First pass: use ping detail IPs (instant, no network), track servers needing DNS
	type dnsPending struct {
		index int
		host  string
	}
	var needDNS []dnsPending
	pingIPs := make(map[int]string, len(srvs))        // index -> public IP from ping
	pingPrivateIPs := make(map[int]string, len(srvs)) // index -> private IP from ping

	for i, srv := range srvs {
		if ping, ok := pingByServerWithDetail[srv.ID]; ok && ping.PingDetail != nil && ping.PingDetail.IP != "" {
			if isPrivateIP(ping.PingDetail.IP) {
				// Track private IP but still try DNS for public IP
				pingPrivateIPs[i] = ping.PingDetail.IP
			} else {
				pingIPs[i] = ping.PingDetail.IP
				continue
			}
		}
		// No public ping IP — need DNS fallback
		host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "https://"), "http://")
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		needDNS = append(needDNS, dnsPending{index: i, host: host})
	}

	// Parallel DNS only for servers without a public ping IP
	type dnsResult struct {
		index      int
		publicIPs  []string
		privateIPs []string
	}
	dnsResultsByIdx := make(map[int]dnsResult, len(needDNS))

	if len(needDNS) > 0 {
		dnsCh := make(chan dnsResult, len(needDNS))
		var dnsWg sync.WaitGroup
		dnsSem := make(chan struct{}, 50)

		for _, pending := range needDNS {
			dnsWg.Add(1)
			go func(idx int, h string) {
				defer dnsWg.Done()
				dnsSem <- struct{}{}
				defer func() { <-dnsSem }()

				var publicIPs, privateIPs []string
				if resolved, err := net.LookupHost(h); err == nil {
					for _, ip := range resolved {
						if isPrivateIP(ip) {
							privateIPs = append(privateIPs, ip)
						} else {
							publicIPs = append(publicIPs, ip)
						}
					}
				}
				dnsCh <- dnsResult{index: idx, publicIPs: publicIPs, privateIPs: privateIPs}
			}(pending.index, pending.host)
		}
		dnsWg.Wait()
		close(dnsCh)

		for r := range dnsCh {
			dnsResultsByIdx[r.index] = r
		}
	}

	// Assemble server data from ping IPs + DNS fallback, tracking both public and private
	allData := make([]serverData, 0, len(srvs))
	allPublicIPs := make(map[string]bool)
	allPrivateIPs := make(map[string]bool)

	for i, srv := range srvs {
		ipSet := make(map[string]bool)

		// Add public IPs
		if ip, ok := pingIPs[i]; ok {
			ipSet[ip] = true
		}
		if dnsRes, ok := dnsResultsByIdx[i]; ok {
			for _, ip := range dnsRes.publicIPs {
				ipSet[ip] = true
			}
		}

		// Add private IPs
		if ip, ok := pingPrivateIPs[i]; ok {
			ipSet[ip] = true
		}
		if dnsRes, ok := dnsResultsByIdx[i]; ok {
			for _, ip := range dnsRes.privateIPs {
				ipSet[ip] = true
			}
		}

		if len(ipSet) == 0 {
			continue
		}

		ips := make([]string, 0, len(ipSet))
		for ip := range ipSet {
			ips = append(ips, ip)
			if isPrivateIP(ip) {
				allPrivateIPs[ip] = true
			} else {
				allPublicIPs[ip] = true
			}
		}

		status := "unknown"
		if ping, ok := pingByServer[srv.ID]; ok {
			if ping.Error != "" {
				status = "down"
			} else if ping.StatusCode == srv.ExpectedStatusCode {
				status = "up"
			} else {
				status = "down"
			}
		}

		allData = append(allData, serverData{srv: &srv, ips: ips, status: status})
	}

	// Phase 3: Geo-lookup all unique public IPs in parallel
	geoResults := make(map[string]*GeoResult)
	var geoMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for ip := range allPublicIPs {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if geo, err := lookupGeo(ip); err == nil {
				geoMu.Lock()
				geoResults[ip] = geo
				geoMu.Unlock()
			} else {
				log.Printf("Geo lookup failed for %s: %v", ip, err)
			}
		}(ip)
	}
	wg.Wait()

	// Phase 4: Assemble deduplicated response
	servers := make([]ServerGeo, 0, len(allData))
	for _, sd := range allData {
		servers = append(servers, ServerGeo{
			ServerID: sd.srv.ID,
			URL:      sd.srv.URL,
			IPs:      sd.ips,
			Status:   sd.status,
		})
	}

	geo := make(map[string]GeoResult, len(geoResults))
	for ip, g := range geoResults {
		geo[ip] = *g
	}

	// Collect local IPs as a sorted list
	localIPs := make([]string, 0, len(allPrivateIPs))
	for ip := range allPrivateIPs {
		localIPs = append(localIPs, ip)
	}
	sort.Strings(localIPs)

	resp := &ServerGeoResponse{
		Servers:  servers,
		Geo:      geo,
		LocalIPs: localIPs,
	}

	geoResponseCacheMu.Lock()
	geoResponseCache = resp
	geoResponseFingerprint = fingerprint
	geoResponseCacheAt = time.Now()
	geoResponseCacheMu.Unlock()

	log.Printf("Geo response cache rebuilt in %v (public IPs: %d, local IPs: %d)",
		time.Since(startTime), len(allPublicIPs), len(allPrivateIPs))
}

// GetServersGeo returns all active servers with their IP and geolocation data.
// Response is served from a background-maintained cache for performance.
// If cache is empty, triggers a background rebuild and returns empty response or waits briefly.
func GetServersGeo(c *gin.Context) {
	// Serve from cache if available
	geoResponseCacheMu.RLock()
	if geoResponseCache != nil {
		resp := geoResponseCache
		geoResponseCacheMu.RUnlock()
		c.JSON(200, resp)
		return
	}
	geoResponseCacheMu.RUnlock()

	// Cache is empty - trigger rebuild and wait briefly with timeout
	done := make(chan struct{})
	go func() {
		rebuildGeoResponseCache()
		close(done)
	}()

	// Wait up to 2 seconds for rebuild to complete
	select {
	case <-done:
		// Rebuild completed, serve the result
		geoResponseCacheMu.RLock()
		resp := geoResponseCache
		geoResponseCacheMu.RUnlock()
		if resp != nil {
			c.JSON(200, resp)
		} else {
			c.JSON(200, ServerGeoResponse{
				Servers:  []ServerGeo{},
				Geo:      map[string]GeoResult{},
				LocalIPs: []string{},
			})
		}
	case <-time.After(2 * time.Second):
		// Timeout - return empty response, rebuild continues in background
		log.Println("Geo cache rebuild timeout, returning empty response")
		c.JSON(200, ServerGeoResponse{
			Servers:  []ServerGeo{},
			Geo:      map[string]GeoResult{},
			LocalIPs: []string{},
		})
	}
}
