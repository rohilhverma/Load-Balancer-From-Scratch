package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
	// "net/http/httputil"
	// "net/url"
)

type Backend struct {
	address    string
	port       int
	status     atomic.Bool
	failHealth atomic.Bool
}

type RoutingImplementation interface {
	rerouter(w http.ResponseWriter, r *http.Request)
}

type BackendImplementation interface {
	backendHit(b *Backend) http.HandlerFunc
	healthCheck(b *Backend) http.HandlerFunc
	failHealth(b *Backend) http.HandlerFunc
}

var (
	serverList = []*Backend{
		{address: "http://localhost:8082", port: 8082},
		{address: "http://localhost:8083", port: 8083},
		{address: "http://localhost:8084", port: 8084},
		{address: "http://localhost:8085", port: 8085},
		{address: "http://localhost:8086", port: 8086},
	}
	serverFailures = make([]atomic.Int64, len(serverList))
	
	rr = RoundRobin{}
	lc = LeastConnections{}
	ip = IPHash{}

	client = &http.Client{Timeout: 4 * time.Second}
)

func BackendServer(b *Backend) {
	backendServer := http.NewServeMux()
	backendServer.HandleFunc("/", ip.backendHit(b))
	backendServer.HandleFunc("/health", ip.healthCheck(b)) // endpoint to test status of servers
backendServer.HandleFunc("/fail", ip.failHealth(b))    // endpoint to trigger servers failing
	fmt.Printf("Backend Server Starting - %d\n", b.port)   //

	err := http.ListenAndServe(":"+strconv.Itoa(b.port), backendServer)
	if err != nil {
		log.Fatal("Backend Server Start Failed")
	}
}

func main() {
	for _, backend := range serverList {
		backend.status.Store(true)
	}

	for server := range serverList {
		go BackendServer(serverList[server])
	}
	go func() {
		for {
			time.Sleep(5 * time.Second)
			for backend := range serverList {
				go healthCheck(backend)
			}
		}

	}()

	http.HandleFunc("/", ip.rerouter)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func healthCheck(indx int) {
	backend := serverList[indx]
	healthAddr := backend.address + "/health"
	response, err := client.Get(healthAddr)
	healthy := err == nil && response != nil &&
		response.StatusCode >= http.StatusOK &&
		response.StatusCode < 210
	if response != nil {
		response.Body.Close()
	}

	if healthy {
		serverFailures[indx].Swap(0)
		backend.status.Swap(true)
		return
	}
	if serverFailures[indx].Add(1) >= 3 {
		backend.status.Store(false)
	}
}
