package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
	// "time"
)

type LeastConnections struct{}

var (
	// set up 3 channels to recieve gouroutines, probably save them to a dict
	connections = make([]atomic.Int64, 5)
	loadMu      sync.Mutex
	// channels = []chan int{make(chan int), make(chan int), make(chan int), make(chan int), make(chan int)}
)

//run 3 gouroutines to run forever to solely handle requests, use a time.sleep to
// imulate more intensive requests

func (lc LeastConnections) rerouter(w http.ResponseWriter, r *http.Request) {
	loadMu.Lock()
	indx := smallestLoad()
	connections[indx].Add(1)
	loadMu.Unlock()
	backend, err := url.Parse(serverTable[int64(indx)])
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

func (lc LeastConnections) backendHit(port int) http.HandlerFunc {
	v := port - 8082
	return func(http.ResponseWriter, *http.Request) {
        t := rand.IntN(10)+1
        fmt.Println(t, " seconds to compelte request")
		time.Sleep(time.Duration(t) * time.Second)
		fmt.Println("Request Handled")
		connections[v].Add(-1)
    }
}

func smallestLoad() int {
	m := connections[0].Load()
	i := 0
	for x := 1; x < len(connections); x++ {
		curr := connections[x].Load()
		if curr < m {
			m = curr
			i = x
		}
	}
	return i
}
