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

type ServerGeo struct {
	ServerID uint       `json:"server_id"`
	URL      string     `json:"url"`
	IP       string     `json:"ip"`
	Geo      *GeoResult `json:"geo,omitempty"`
	Status   string     `json:"status"`
}

var (
	geoCache    = make(map[string]*GeoResult)
	geoCacheMu  sync.RWMutex
	geoCacheTTL = 24 * time.Hour
	geoCacheTS  = make(map[string]time.Time)
)

// geoProviders are tried in order until one succeeds
var geoProviders = []func(string) (*GeoResult, error){
	lookupIPWhois,
	lookupIPAPI,
}

func lookupGeo(ip string) (*GeoResult, error) {
	geoCacheMu.RLock()
	if cached, ok := geoCache[ip]; ok {
		if time.Since(geoCacheTS[ip]) < geoCacheTTL {
			geoCacheMu.RUnlock()
			return cached, nil
		}
	}
	geoCacheMu.RUnlock()

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
	geoCacheMu.Lock()
	geoCache[ip] = result
	geoCacheTS[ip] = time.Now()
	geoCacheMu.Unlock()
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

	results := make([]ServerGeo, 0, len(servers))

	for _, srv := range servers {
		var ip string

		// Try to get IP from latest ping detail
		var ping PingResult
		if err := db.Where("server_id = ? AND detail_id IS NOT NULL", srv.ID).
			Preload("PingDetail").
			Order("timestamp DESC").
			First(&ping).Error; err == nil && ping.PingDetail != nil {
			ip = ping.PingDetail.IP
		}

		// Fallback: resolve the domain via DNS
		if ip == "" {
			host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "https://"), "http://")
			if resolved, err := net.LookupHost(host); err == nil && len(resolved) > 0 {
				ip = resolved[0]
			}
		}

		if ip == "" {
			continue
		}

		// Get latest ping for status (any ping, not just ones with detail)
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

		sg := ServerGeo{
			ServerID: srv.ID,
			URL:      srv.URL,
			IP:       ip,
			Status:   status,
		}

		geo, err := lookupGeo(ip)
		if err != nil {
			log.Printf("Geo lookup failed for %s (%s): %v", srv.URL, ip, err)
		} else {
			sg.Geo = geo
		}

		results = append(results, sg)
	}

	c.JSON(200, results)
}
