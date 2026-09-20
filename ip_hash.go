package main

import (
	"fmt"
	"hash/fnv"
	"net"
	"net/http"
	"strings"
)

type IPHash struct{}

func (ip IPHash) rerouter(w http.ResponseWriter, r *http.Request) {
	client := clientIP(r)
	proxyRequest(w, r, func(exclude *Backend) (*Backend, func(), error) {
		backend, err := hashBackend(client, exclude)
		return backend, func() {}, err
	})
}

func (ip IPHash) backendHit(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Backend-Port", fmt.Sprint(b.port))
	}
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

func (ip IPHash) reverseHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(false)
		w.WriteHeader(http.StatusNoContent)
	}
}



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

func hashBackend(ip string, exclude *Backend) (*Backend, error) {
	var m uint64
	var selected *Backend
	for _, server := range serverList {
		if server != exclude && server.status.Load() {
			curr := redevHashing(ip, server.address)
			if curr >= m {
				m = curr
				selected = server
			}
		}
	}

	if selected == nil {
		return nil, NoActiveServers
	}
	return selected, nil
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
