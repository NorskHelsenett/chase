package servers

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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

type IPInfo struct {
	IP  string     `json:"ip"`
	Geo *GeoResult `json:"geo,omitempty"`
}

type ServerGeo struct {
	ServerID uint     `json:"server_id"`
	URL      string   `json:"url"`
	IPs      []IPInfo `json:"ips"`
	Status   string   `json:"status"`
}

var (
	geoMemCache   = make(map[string]*GeoResult)
	geoMemCacheMu sync.RWMutex
	geoCacheTTL   = 7 * 24 * time.Hour
)

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

// GetServersGeo returns all active servers with their IP and geolocation data
func GetServersGeo(c *gin.Context) {
	db := database.GetDB()

	var servers []Server
	if err := db.Where("active = ?", true).Find(&servers).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch servers"})
		return
	}

	// Phase 1: Collect all unique public IPs across all servers (and per-server data)
	type serverData struct {
		srv    Server
		ipSet  map[string]bool
		status string
	}
	allData := make([]serverData, 0, len(servers))
	allIPs := make(map[string]bool) // global dedup for geo lookups

	for _, srv := range servers {
		ipSet := make(map[string]bool)

		// IP from latest ping detail
		var ping PingResult
		if err := db.Where("server_id = ? AND detail_id IS NOT NULL", srv.ID).
			Preload("PingDetail").
			Order("timestamp DESC").
			First(&ping).Error; err == nil && ping.PingDetail != nil && ping.PingDetail.IP != "" {
			if !isPrivateIP(ping.PingDetail.IP) {
				ipSet[ping.PingDetail.IP] = true
			}
		}

		// All IPs from DNS resolution
		host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "https://"), "http://")
		if idx := strings.Index(host, "/"); idx != -1 {
			host = host[:idx]
		}
		if resolved, err := net.LookupHost(host); err == nil {
			for _, ip := range resolved {
				if !isPrivateIP(ip) {
					ipSet[ip] = true
				}
			}
		}

		if len(ipSet) == 0 {
			continue
		}

		for ip := range ipSet {
			allIPs[ip] = true
		}

		// Determine status from latest ping
		status := "unknown"
		var latestPing PingResult
		if err := db.Where("server_id = ?", srv.ID).
			Order("timestamp DESC").
			First(&latestPing).Error; err == nil {
			if latestPing.Error != "" {
				status = "down"
			} else if latestPing.StatusCode == srv.ExpectedStatusCode {
				status = "up"
			} else {
				status = "down"
			}
		}

		allData = append(allData, serverData{srv: srv, ipSet: ipSet, status: status})
	}

	// Phase 2: Geo-lookup all unique IPs in parallel
	geoResults := make(map[string]*GeoResult)
	var geoMu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // limit concurrency to 10

	for ip := range allIPs {
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

	// Phase 3: Assemble results
	results := make([]ServerGeo, 0, len(allData))
	for _, sd := range allData {
		ips := make([]IPInfo, 0, len(sd.ipSet))
		for ip := range sd.ipSet {
			info := IPInfo{IP: ip}
			if geo, ok := geoResults[ip]; ok {
				info.Geo = geo
			}
			ips = append(ips, info)
		}

		results = append(results, ServerGeo{
			ServerID: sd.srv.ID,
			URL:      sd.srv.URL,
			IPs:      ips,
			Status:   sd.status,
		})
	}

	c.JSON(200, results)
}
