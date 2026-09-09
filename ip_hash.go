package main

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)


type IPHash struct{}

func (ip IPHash) rerouter(w http.ResponseWriter, r *http.Request){ 
	backend, err := url.Parse(hashFunction(r.Header.Get("X-Forwarded-For")))
	fmt.Printf("IP ", r.Header.Get("X-Forwarded-For")," hashed to ",backend)
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

func (ip IPHash) backendHit(port int) http.HandlerFunc{
	return func(http.ResponseWriter,*http.Request){fmt.Printf("Request Handled")}
}

func hashFunction(ip string) string {
	hasher := fnv.New64a()
	hasher.Write([]byte(ip))
	hashNumber := hasher.Sum64()
	return serverTable[int64(hashNumber % 5)]
}