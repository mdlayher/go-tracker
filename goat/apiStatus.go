package goat

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"sync/atomic"
)

// serverStatus represents a struct to be serialized, containing information about the system running goat
type serverStatus struct {
	PID          int       `json:"pid"`
	Hostname     string    `json:"hostname"`
	Platform     string    `json:"platform"`
	Architecture string    `json:"architecture"`
	NumCPU       int       `json:"numCpu"`
	NumGoroutine int       `json:"numGoroutine"`
	MemoryMB     float64   `json:"memoryMb"`
	HTTP         httpStats `json:"http"`
	UDP          udpStats  `json:"udp"`
}

// httpStats represents statistics regarding HTTP server
type httpStats struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// udpStats represents statistics regarding UDP server
type udpStats struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// getServerStatus represents a tracker status request
func getServerStatus() serverStatus {
	// Get system hostname
	var hostname string
	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err.Error())
		return serverStatus{}
	}

	// Get current memory profile
	mem := &runtime.MemStats{}
	runtime.ReadMemStats(mem)

	// Report memory usage in MB
	memMb := float64((float64(mem.Alloc) / 1000) / 1000)

	// HTTP status
	httpStatus := httpStats{
		atomic.LoadInt64(&static.HTTP.Current),
		atomic.LoadInt64(&static.HTTP.Total),
	}

	// UDP status
	udpStatus := udpStats{
		atomic.LoadInt64(&static.UDP.Current),
		atomic.LoadInt64(&static.UDP.Total),
	}

	// Build status struct
	status := serverStatus{
		os.Getpid(),
		hostname,
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		runtime.NumGoroutine(),
		memMb,
		httpStatus,
		udpStatus,
	}

	// Return status struct
	return status
}

// GetStatusJSON returns a JSON representation of server status
func getStatusJSON(resChan chan []byte) {
	// Marshal into JSON from request
	res, err := json.Marshal(getServerStatus())
	if err != nil {
		log.Println(err.Error())
		resChan <- nil
	}

	// Return status
	resChan <- res
}

// PrintStatusBanner logs the startup status banner
func printStatusBanner() {
	// Grab initial server status
	stat := getServerStatus()
	if stat == (serverStatus{}) {
		log.Println("Could not print startup status banner")
		return
	}

	// Startup banner
	log.Printf("%s - %s_%s (%d CPU) [pid: %d]", stat.Hostname, stat.Platform, stat.Architecture, stat.NumCPU, stat.PID)
}

// PrintCurrentStatus logs the regular status check banner
func printCurrentStatus() {
	// Grab server status
	stat := getServerStatus()
	if stat == (serverStatus{}) {
		log.Println("Could not print current status")
		return
	}

	// Regular status banner
	log.Printf("status - [goroutines: %d] [memory: %02.3f MB]", stat.NumGoroutine, stat.MemoryMB)

	// HTTP stats
	if static.Config.HTTP {
		log.Printf("  http - [current: %d] [total: %d]", stat.HTTP.Current, stat.HTTP.Total)

		// Reset current HTTP counter
		atomic.StoreInt64(&static.HTTP.Current, 0)
	}

	// UDP stats
	if static.Config.UDP {
		log.Printf("   udp - [current: %d] [total: %d]", stat.UDP.Current, stat.UDP.Total)

		// Reset current UDP counter
		atomic.StoreInt64(&static.UDP.Current, 0)
	}
}
