package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	// "net/http/httputil"
	// "net/url"
)

type RoutingImplementation interface {
	rerouter(w http.ResponseWriter, r *http.Request)
}

type BackendImplementation interface {
	backendHit(port int) http.HandlerFunc
}

var (
	serverTable = map[int64]string{
		0: "http://localhost:8082",
		1: "http://localhost:8083",
		2: "http://localhost:8084",
		3: "http://localhost:8085",
		4: "http://localhost:8086",
	}

	rr = RoundRobin{}
	lc = LeastConnections{}
	ip = IPHash{}
)

func BackendServer(port int) {
	backendServer := http.NewServeMux()
	backendServer.HandleFunc("/", ip.backendHit(port))
	fmt.Printf("Backend Server Starting - %d\n", port)

	if err := http.ListenAndServe(":" + strconv.Itoa(port), backendServer); err != nil {
		log.Fatal("Backend Server Start Failed")
	}
}

func main() {
	for _, port := range []int{8082, 8083, 8084, 8085, 8086} {
		go BackendServer(port)
	}

	http.HandleFunc("/", ip.rerouter)
	log.Fatal(http.ListenAndServe(":8081", nil))
}
