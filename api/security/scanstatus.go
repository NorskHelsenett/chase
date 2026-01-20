package security

import (
	"sync"
	"time"
)

type ScanStatus struct {
	ID          string    `json:"id,omitempty"`
	State       string    `json:"state"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
	Error       string    `json:"error,omitempty"`
}

var (
	scanStatusMu       sync.RWMutex
	scanStatusByServer = make(map[string]*ScanStatus)
)

func getScanStatus(serverURL string) *ScanStatus {
	scanStatusMu.RLock()
	defer scanStatusMu.RUnlock()
	status, ok := scanStatusByServer[serverURL]
	if !ok || status == nil {
		return nil
	}
	copy := *status
	return &copy
}

func markScanRunning(serverURL string) *ScanStatus {
	scanStatusMu.Lock()
	defer scanStatusMu.Unlock()
	if existing, ok := scanStatusByServer[serverURL]; ok && existing != nil && existing.State == "running" {
		copy := *existing
		return &copy
	}
	status := &ScanStatus{
		ID:        generateJobID(),
		State:     "running",
		StartedAt: time.Now(),
	}
	scanStatusByServer[serverURL] = status
	copy := *status
	return &copy
}

func markScanFailed(serverURL string, err error) {
	scanStatusMu.Lock()
	defer scanStatusMu.Unlock()
	status := &ScanStatus{
		State:       "failed",
		CompletedAt: time.Now(),
	}
	if err != nil {
		status.Error = sanitizeScanError(err.Error())
	}
	if existing, ok := scanStatusByServer[serverURL]; ok && existing != nil {
		status.ID = existing.ID
		status.StartedAt = existing.StartedAt
	}
	scanStatusByServer[serverURL] = status
}

func clearScanStatus(serverURL string) {
	scanStatusMu.Lock()
	defer scanStatusMu.Unlock()
	delete(scanStatusByServer, serverURL)
}
