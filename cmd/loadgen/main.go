package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type config struct {
	url         string
	requests    int
	concurrency int
	clients     int
}

type result struct {
	latency     time.Duration
	statusCode  int
	backendPort string
	err         error
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.url, "url", "http://localhost:8081/", "load balancer URL")
	flag.IntVar(&cfg.requests, "requests", 10000, "total number of requests")
	flag.IntVar(&cfg.concurrency, "concurrency", 50, "number of concurrent workers")
	flag.IntVar(&cfg.clients, "clients", 250, "number of simulated client IPs")
	flag.Parse()

	if err := validateConfig(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	run(cfg)
}

func validateConfig(cfg config) error {
	if cfg.url == "" {
		return fmt.Errorf("url must not be empty")
	}
	if cfg.requests <= 0 {
		return fmt.Errorf("requests must be positive")
	}
	if cfg.concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive")
	}
	if cfg.clients <= 0 {
		return fmt.Errorf("clients must be positive")
	}
	return nil
}

func run(cfg config) {
	transport := &http.Transport{
		MaxIdleConns:        cfg.concurrency,
		MaxIdleConnsPerHost: cfg.concurrency,
		MaxConnsPerHost:     cfg.concurrency,
		IdleConnTimeout:     30 * time.Second,
	}
	defer transport.CloseIdleConnections()

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	jobs := make(chan int)
	results := make(chan result, cfg.concurrency)

	var workers sync.WaitGroup
	workers.Add(cfg.concurrency)
	for range cfg.concurrency {
		go func() {
			defer workers.Done()
			for requestNumber := range jobs {
				results <- makeRequest(client, cfg.url, requestNumber%cfg.clients)
			}
		}()
	}

	start := time.Now()
	go func() {
		for i := range cfg.requests {
			jobs <- i
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()

	latencies := make([]time.Duration, 0, cfg.requests)
	statusCounts := make(map[int]int)
	backendCounts := make(map[string]int)
	successes := 0
	failures := 0

	for res := range results {
		latencies = append(latencies, res.latency)
		if res.err != nil {
			failures++
			continue
		}
		statusCounts[res.statusCode]++
		if res.backendPort != "" {
			backendCounts[res.backendPort]++
		}
		if res.statusCode >= 200 && res.statusCode < 400 {
			successes++
		} else {
			failures++
		}
	}
	elapsed := time.Since(start)

	fmt.Printf("Requests:    %d\n", cfg.requests)
	fmt.Printf("Concurrency: %d\n", cfg.concurrency)
	fmt.Printf("Elapsed:     %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("RPS:         %.2f\n", float64(cfg.requests)/elapsed.Seconds())
	fmt.Printf("Success:     %d\n", successes)
	fmt.Printf("Failure:     %d\n", failures)
	fmt.Printf("Latency:     avg=%s p50=%s p95=%s p99=%s\n",
		average(latencies).Round(time.Microsecond),
		percentile(latencies, 50).Round(time.Microsecond),
		percentile(latencies, 95).Round(time.Microsecond),
		percentile(latencies, 99).Round(time.Microsecond),
	)
	printIntCounts("Status codes", statusCounts)
	printStringCounts("Backends", backendCounts)
}

func makeRequest(client *http.Client, target string, clientNumber int) result {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return result{err: err}
	}
	req.Header.Set("X-Forwarded-For", simulatedIP(clientNumber))

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return result{latency: time.Since(start), err: err}
	}
	_, copyErr := io.Copy(io.Discard, resp.Body)
	closeErr := resp.Body.Close()
	latency := time.Since(start)
	if copyErr != nil {
		return result{latency: latency, err: copyErr}
	}
	if closeErr != nil {
		return result{latency: latency, err: closeErr}
	}
	return result{
		latency:     latency,
		statusCode:  resp.StatusCode,
		backendPort: resp.Header.Get("X-Backend-Port"),
	}
}

func simulatedIP(n int) string {
	return fmt.Sprintf("10.%d.%d.%d", (n>>16)&255, (n>>8)&255, n&255)
}

func average(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	var total time.Duration
	for _, value := range values {
		total += value
	}
	return total / time.Duration(len(values))
}

func percentile(values []time.Duration, percent int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := (percent*len(sorted) + 99) / 100
	if index < 1 {
		index = 1
	}
	if index > len(sorted) {
		index = len(sorted)
	}
	return sorted[index-1]
}

func printIntCounts(label string, counts map[int]int) {
	keys := make([]int, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	fmt.Printf("%s:\n", label)
	for _, key := range keys {
		fmt.Printf("  %d: %d\n", key, counts[key])
	}
}

func printStringCounts(label string, counts map[string]int) {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Printf("%s:\n", label)
	for _, key := range keys {
		fmt.Printf("  %s: %d\n", key, counts[key])
	}
}
