package main

import (
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
)

var (
	count           atomic.Int64
	NoActiveServers = errors.New("No Active Servers")
)

type RoundRobin struct{}

func (rr RoundRobin) rerouter(w http.ResponseWriter, r *http.Request) {
	proxyRequest(w, r, func(exclude *Backend) (*Backend, func(), error) {
		backend, err := atomicCounterExcluding(exclude)
		return backend, func() {}, err
	})
}

func (rr RoundRobin) backendHit(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Backend-Port", fmt.Sprint(b.port))
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

func (rr RoundRobin) reverseHealth(b *Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b.failHealth.Store(false)
		w.WriteHeader(http.StatusNoContent)
	}
}

func atomicCounter() (*Backend, error) {
	return atomicCounterExcluding(nil)
}

func atomicCounterExcluding(exclude *Backend) (*Backend, error) {
	total := len(serverList)
	if total == 0 {
		return nil, NoActiveServers
	}

	for range total {
		i := int((count.Add(1) - 1) % int64(total))
		backend := serverList[i]
		//inactive servers skipped
		if backend != exclude && backend.status.Load() {
			return backend, nil
		}
	}
	// all subsequent servers failed
	return nil, NoActiveServers
}
