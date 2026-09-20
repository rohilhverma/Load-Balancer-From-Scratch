package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Backend struct {
	address    string
	port       int
	status     atomic.Bool
	failHealth atomic.Bool
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

	ip = IPHash{}

	client = &http.Client{Timeout: 4 * time.Second}
)

func BackendServer(b *Backend) error {
	backendServer := http.NewServeMux()
	backendServer.HandleFunc("/", ip.backendHit(b))
	backendServer.HandleFunc("/health", ip.healthCheck(b))    // endpoint to test status of servers
	backendServer.HandleFunc("/fail", ip.failHealth(b))       // endpoint to trigger servers failing
	backendServer.HandleFunc("/reverse", ip.reverseHealth(b)) // endpoint to undo /fail so the server can recover
	fmt.Printf("Backend Server Starting - %d\n", b.port)
	return http.ListenAndServe(":"+strconv.Itoa(b.port), backendServer)
}

func main() {
	mode := flag.String("mode", "all", "run mode: all, loadbalancer, or backend")
	listen := flag.String("listen", ":8081", "load balancer listen address")
	port := flag.Int("port", 8082, "backend listen port")
	backends := flag.String("backends", os.Getenv("BACKENDS"), "comma-separated backend URLs (or set BACKENDS)")
	flag.Parse()

	var err error
	switch *mode {
	case "all":
		err = runAll(*listen)
	case "loadbalancer":
		if *backends == "" {
			log.Fatal("loadbalancer mode requires -backends or BACKENDS")
		}
		if err = configureBackends(*backends); err == nil {
			err = runLoadBalancer(*listen)
		}
	case "backend":
		if *port < 1 || *port > 65535 {
			log.Fatalf("invalid backend port %d", *port)
		}
		err = BackendServer(&Backend{port: *port})
	default:
		log.Fatalf("unknown mode %q (use all, loadbalancer, or backend)", *mode)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func runAll(listen string) error {
	for _, backend := range serverList {
		backend.status.Store(true)
	}

	for _, backend := range serverList {
		go func() {
			if err := BackendServer(backend); err != nil {
				log.Printf("backend %d stopped: %v", backend.port, err)
			}
		}()
	}
	return runLoadBalancer(listen)
}

func runLoadBalancer(listen string) error {
	for _, backend := range serverList {
		backend.status.Store(true)
	}
	startHealthChecks()

	mux := http.NewServeMux()
	mux.HandleFunc("/", ip.rerouter)
	log.Printf("Load balancer listening on %s", listen)
	return http.ListenAndServe(listen, mux)
}

func startHealthChecks() {
	go func() {
		for {
			for backend := range serverList {
				go healthCheck(backend)
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func configureBackends(raw string) error {
	addresses := strings.Split(raw, ",")
	configured := make([]*Backend, 0, len(addresses))
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		parsed, err := url.ParseRequestURI(address)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid backend URL %q", address)
		}
		port, err := strconv.Atoi(parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("backend URL %q must include a valid port", address)
		}
		configured = append(configured, &Backend{address: strings.TrimRight(address, "/"), port: port})
	}
	if len(configured) == 0 {
		return fmt.Errorf("at least one backend URL is required")
	}

	serverList = configured
	serverFailures = make([]atomic.Int64, len(serverList))
	connections = make([]atomic.Int64, len(serverList))
	count.Store(0)
	return nil
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
