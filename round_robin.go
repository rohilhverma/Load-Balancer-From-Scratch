
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

var (
	count           atomic.Int64
	NoActiveServers = errors.New("No Active Servers")
)

type RoundRobin struct{}

func (rr RoundRobin) rerouter(w http.ResponseWriter, r *http.Request) {
	backend, err := atomicCounter()
	if err != nil {
		if errors.Is(err, NoActiveServers) {
			http.Error(w, "No Healthy Backend Available", http.StatusServiceUnavailable)
		}
		return
	}
	addr, err := url.Parse(backend.address)
	if err != nil {
		log.Fatal(("Unable to find new server"))
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(s *httputil.ProxyRequest) {
			s.SetURL(addr)
		},
	}
	proxy.ServeHTTP(w, r)
	fmt.Println("Proxied to", backend.address)
}

func (rr RoundRobin) backendHit(b *Backend) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {
		fmt.Println("Handling Request at Port", b.port)
	}
}

func (rr RoundRobin) healthCheck(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if b.failHealth.Load() {
			http.Error(w, "Backend unhealthy", http.StatusServiceUnavailable)
			return
		} 
		w.WriteHeader(http.StatusOK)
	}
}

func (rr RoundRobin) failHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}
}

func atomicCounter() (*Backend,error) {
	total := len(serverList)
	if total == 0 {
		return nil, NoActiveServers
	}

	for range total {
		i := int((count.Add(1) - 1) % int64(total))
		backend := serverList[i]
		//inactive servers skipped
		if backend.status.Load() {
			return backend, nil
		}
	}
	// all subsequent servers failed
	return nil, NoActiveServers
}
