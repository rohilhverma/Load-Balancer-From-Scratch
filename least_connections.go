package main

// need to figure out how to efectively log/track the load part. 


import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type LeastConnections struct{}

var (
	connections = make([]atomic.Int64, len(serverList))
	loadMu      sync.Mutex
)

//run 3 gouroutines to run forever to solely handle requests, use a time.sleep to
// imulate more intensive requests

func (lc LeastConnections) rerouter(w http.ResponseWriter, r *http.Request) {
	indx,err := smallestLoad()
	if err != nil {
		if errors.Is(err, NoActiveServers) {
			http.Error(w, "No Healthy Backend Available", http.StatusServiceUnavailable)
		}
		return
	}
	defer connections[indx].Add(-1)
	backend, err := url.Parse(serverList[indx].address)
	if err != nil {
		log.Fatal(("Unable to find new server"))
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(s *httputil.ProxyRequest) {
			s.SetURL(backend)
		},
	}
	proxy.ServeHTTP(w, r)
	fmt.Println("Proxied", backend)
}

func (lc LeastConnections) backendHit(b *Backend) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {
		t := rand.IntN(10) + 1
		fmt.Println(t, " seconds to compelte request")
		time.Sleep(time.Duration(t) * time.Second)
		fmt.Println("Request Handled")
	}
}

// smallestLoad picks the least-loaded live server and counts the new request
// against it in one critical section, so two concurrent requests can't both
// see the same server as idle.
func smallestLoad() (int,error) {
	loadMu.Lock()
	defer loadMu.Unlock()
	i := -1
	m := int64(math.MaxInt64)
	for x:= range len(serverList) {
		if !serverList[x].status.Load(){
			continue
		}
		curr := connections[x].Load()
		
		if curr <= m {
			m = curr
			i = x
		} 
	}
	if i == -1{
		return -1,NoActiveServers
	}
	connections[i].Add(1)
	return i,nil
}

func (lc LeastConnections) healthCheck(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if b.failHealth.Load() {
			http.Error(w, "Backend unhealthy", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (lc LeastConnections) failHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}
}
