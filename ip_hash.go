package main
// need to simulate other IPs
import (
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type IPHash struct{}

func (ip IPHash) rerouter(w http.ResponseWriter, r *http.Request) {
	client := clientIP(r)
	addr, err := hashFunction(client)
	if err != nil {
		if errors.Is(err, NoActiveServers) {
			http.Error(w, "No Healthy Backend Available", http.StatusServiceUnavailable)
		}
		return
	}
	backend, err := url.Parse(addr)
	if err != nil {
		log.Fatal(("Unable to find new server"))
	}
	fmt.Printf("IP %s hashed to %v\n", client, backend)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(s *httputil.ProxyRequest) {
			s.SetURL(backend)
		},
	}
	proxy.ServeHTTP(w, r)
	fmt.Println("Proxied to", backend)
}

func (ip IPHash) backendHit(b *Backend) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) { fmt.Println("Request Handled") }
}

func (ip IPHash) healthCheck(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if b.failHealth.Load() {
			http.Error(w, "backend unhealthy", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func (ip IPHash) failHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}
}

// clientIP returns the original client from X-Forwarded-For, which can hold a
// "client, proxy1, proxy2" chain, falling back to the direct peer address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		first, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(first)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func hashFunction(ip string) (string, error) {
	var m uint64
	addr := ""
	for _, server := range serverList {
		if server.status.Load() {
			curr := redevHashing(ip, server.address)
			if curr >= m {
				m = curr
				addr = server.address
			}
		}
	}

	if addr == "" {
		return "", NoActiveServers
	}
	return addr,nil
}

func redevHashing(ip string, serveraddr string) uint64 {
	hasher := fnv.New64a()
	hasher.Write([]byte(ip))
	hasher.Write([]byte{0})
	hasher.Write([]byte(serveraddr))
	return mix64(hasher.Sum64())
}

// mix64 is the splitmix64 finalizer. FNV-1a barely scrambles its last input
// bytes, and the server addresses only differ in their last digit, so without
// this some servers win far more often than others.
func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}