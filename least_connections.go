package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LeastConnections struct{}

var (
	connections = make([]atomic.Int64, len(serverList))
	loadMu      sync.Mutex
)

func (lc LeastConnections) rerouter(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, r, func(exclude *Backend) (*Backend, func(), error) {
		indx, err := smallestLoadExcluding(exclude)
		if err != nil {
			return nil, func() {}, err
		}
		return serverList[indx], func() { connections[indx].Add(-1) }, nil
	})
}

func (lc LeastConnections) backendHit(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Backend-Port", fmt.Sprint(b.port))
		t := rand.IntN(10) + 1
		time.Sleep(time.Duration(t) * time.Second)

	}
}



func smallestLoadExcluding(exclude *Backend) (int, error) {
	loadMu.Lock()
	defer loadMu.Unlock()
	i := -1
	m := int64(math.MaxInt64)
	for x := range len(serverList) {
		if serverList[x] == exclude || !serverList[x].status.Load() {
			continue
		}
		curr := connections[x].Load()

		if curr <= m {
			m = curr
			i = x
		}
	}
	if i == -1 {
		return -1, NoActiveServers
	}
	connections[i].Add(1)
	return i, nil
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

func (lc LeastConnections) reverseHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(false)
		w.WriteHeader(http.StatusNoContent)
	}
}
