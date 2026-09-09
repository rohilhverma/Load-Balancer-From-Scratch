package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

var (
	count atomic.Int64
)

type RoundRobin struct{}

func (rr RoundRobin) rerouter(w http.ResponseWriter, r *http.Request) {
	backend, err := url.Parse(atomicCounter())
	if err != nil {
		log.Fatal(("Unable to find new server"))
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(s *httputil.ProxyRequest) {
			s.SetURL(backend)
		},
	}
	proxy.ServeHTTP(w, r)
	fmt.Println("Proxied to", backend)
}

func (rr RoundRobin) backendHit(port int) http.HandlerFunc{
	fmt.Println("Handling Request at Port ", port)
	return func(http.ResponseWriter, *http.Request){fmt.Println("Done")}
	
}	

func atomicCounter() string {
	return serverTable[count.Add(1)%5]
}
