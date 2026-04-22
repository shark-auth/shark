// Package main implements a load-stress baseline for SharkAuth.
//
// # Usage
//
//	go run ./test/stress [flags]
//
// Flags
//
//	-base           Base URL of the SharkAuth server (default: http://localhost:8080)
//	-duration       Test duration (default: 30s)
//	-concurrency    Number of concurrent workers (default: 500)
//	-admin-token    Bearer token for /admin/users endpoint
//	-client-id      OAuth client_id for client_credentials grant
//	-client-secret  OAuth client_secret for client_credentials grant
//
// Outputs a JSON summary to stdout and exits 1 if any assertion fails:
//   - 0 5xx responses
//   - p99 latency < 500ms
//   - throughput > 100 req/s
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Summary is the JSON-serialisable result of a stress run.
type Summary struct {
	DurationSec  float64 `json:"duration_sec"`
	Concurrency  int     `json:"concurrency"`
	TotalReqs    int64   `json:"total_requests"`
	FiveXXCount  int64   `json:"5xx_count"`
	ThroughputRPS float64 `json:"throughput_rps"`
	P50MS        float64 `json:"p50_ms"`
	P95MS        float64 `json:"p95_ms"`
	P99MS        float64 `json:"p99_ms"`
	Passed       bool    `json:"assertions_passed"`
	Failures     []string `json:"assertion_failures,omitempty"`
}

func main() {
	base := flag.String("base", "http://localhost:8080", "Base URL of SharkAuth")
	duration := flag.Duration("duration", 30*time.Second, "Test duration")
	concurrency := flag.Int("concurrency", 500, "Concurrent workers")
	adminToken := flag.String("admin-token", "", "Bearer token for /admin/users")
	clientID := flag.String("client-id", "", "OAuth client_id for CC grant")
	clientSecret := flag.String("client-secret", "", "OAuth client_secret for CC grant")
	flag.Parse()

	endpoints := buildEndpoints(*base, *adminToken, *clientID, *clientSecret)

	var (
		totalReqs   int64
		fiveXXCount int64
		mu          sync.Mutex
		latencies   []float64
	)

	stop := make(chan struct{})
	time.AfterFunc(*duration, func() { close(stop) })

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConnsPerHost: *concurrency + 10,
			MaxConnsPerHost:     *concurrency + 10,
		},
	}

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		workerIdx := i
		go func() {
			defer wg.Done()
			ep := endpoints[workerIdx%len(endpoints)]
			for {
				select {
				case <-stop:
					return
				default:
				}

				t0 := time.Now()
				status := doRequest(client, ep)
				latMS := float64(time.Since(t0).Microseconds()) / 1000.0

				atomic.AddInt64(&totalReqs, 1)
				if status >= 500 {
					atomic.AddInt64(&fiveXXCount, 1)
				}

				mu.Lock()
				latencies = append(latencies, latMS)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start).Seconds()

	sort.Float64s(latencies)
	n := len(latencies)

	p50 := percentile(latencies, n, 50)
	p95 := percentile(latencies, n, 95)
	p99 := percentile(latencies, n, 99)
	tput := float64(totalReqs) / elapsed

	summary := Summary{
		DurationSec:   elapsed,
		Concurrency:   *concurrency,
		TotalReqs:     totalReqs,
		FiveXXCount:   fiveXXCount,
		ThroughputRPS: tput,
		P50MS:         p50,
		P95MS:         p95,
		P99MS:         p99,
	}

	// --- Assertions ---
	var failures []string
	if fiveXXCount != 0 {
		failures = append(failures, fmt.Sprintf("5xx count = %d (want 0)", fiveXXCount))
	}
	if p99 >= 500 {
		failures = append(failures, fmt.Sprintf("p99 = %.1f ms (want < 500 ms)", p99))
	}
	if tput <= 100 {
		failures = append(failures, fmt.Sprintf("throughput = %.1f req/s (want > 100 req/s)", tput))
	}

	summary.Passed = len(failures) == 0
	summary.Failures = failures

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		fmt.Fprintf(os.Stderr, "encode summary: %v\n", err)
		os.Exit(2)
	}

	if !summary.Passed {
		fmt.Fprintln(os.Stderr, "STRESS: assertions FAILED")
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "STRESS: all assertions PASSED")
}

// endpoint describes a single HTTP request template.
type endpoint struct {
	method      string
	url         string
	headers     map[string]string
	body        string
	contentType string
}

// buildEndpoints constructs the three round-robin endpoints.
func buildEndpoints(base, adminToken, clientID, clientSecret string) []endpoint {
	eps := []endpoint{
		// 1. GET /healthz — no auth
		{
			method: "GET",
			url:    base + "/healthz",
		},
	}

	// 2. POST /oauth/token — client_credentials (only if credentials provided)
	if clientID != "" && clientSecret != "" {
		creds := base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
		eps = append(eps, endpoint{
			method: "POST",
			url:    base + "/oauth/token",
			headers: map[string]string{
				"Authorization": "Basic " + creds,
			},
			body:        "grant_type=client_credentials",
			contentType: "application/x-www-form-urlencoded",
		})
	}

	// 3. GET /admin/users — requires admin bearer (only if token provided)
	if adminToken != "" {
		eps = append(eps, endpoint{
			method: "GET",
			url:    base + "/api/v1/admin/users",
			headers: map[string]string{
				"Authorization": "Bearer " + adminToken,
			},
		})
	}

	return eps
}

// doRequest executes an HTTP request and returns the status code.
// Returns 0 on transport/network error (not counted as 5xx).
func doRequest(client *http.Client, ep endpoint) int {
	var bodyReader io.Reader
	if ep.body != "" {
		bodyReader = strings.NewReader(ep.body)
	}

	req, err := http.NewRequest(ep.method, ep.url, bodyReader)
	if err != nil {
		return 0
	}

	if ep.contentType != "" {
		req.Header.Set("Content-Type", ep.contentType)
	}
	for k, v := range ep.headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	resp.Body.Close()
	return resp.StatusCode
}

// percentile returns the p-th percentile (0-100) of a sorted float64 slice.
func percentile(sorted []float64, n int, p float64) float64 {
	if n == 0 {
		return 0
	}
	rank := (p / 100.0) * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi || hi >= n {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
