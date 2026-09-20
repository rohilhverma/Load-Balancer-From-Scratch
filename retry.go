package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)


type backendSelector func(exclude *Backend) (backend *Backend, release func(), err error)

func proxyRequest(w http.ResponseWriter, r *http.Request, selectBackend backendSelector) {
	backend, release, err := selectBackend(nil)

	if err != nil {
		writeProxySelectionError(w, err)
		return
	}
	defer release()
	proxyAttempt(w, r, backend, selectBackend, retryableRequest(r))
}

func proxyAttempt(w http.ResponseWriter, r *http.Request, backend *Backend, selectBackend backendSelector, mayRetry bool) {
	target, err := url.Parse(backend.address)
	if err != nil {
		log.Printf("unvalid backend address %q: %v", backend.address, err)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
		},
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, transportErr error) {
		if !mayRetry {
			log.Printf("proxy failure for %s: %v", backend.address, transportErr)
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}

		next, release, err := selectBackend(backend)

		if err != nil {
			writeProxySelectionError(w, err)
			return
			}
		defer release()

		log.Printf("proxy transport failure for %s; retrying on %s: %v", backend.address, next.address, transportErr)
		proxyAttempt(w, r, next, selectBackend, false)
	}
	proxy.ServeHTTP(w, r)
}



func retryableRequest(r *http.Request) bool {
	if r.Body != nil && r.Body != http.NoBody {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}


func writeProxySelectionError(w http.ResponseWriter, err error) {
	if errors.Is(err, NoActiveServers) {
		http.Error(w, "no backend Available", http.StatusServiceUnavailable)
		return
	}
	log.Printf("backend selection failed: %v", err)
	http.Error(w, "bad gateway", http.StatusBadGateway)
}
