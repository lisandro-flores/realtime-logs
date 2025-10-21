package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type LogEntry struct {
	OrgID     string `json:"org_id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp,omitempty"`
}

type IngestPayload struct {
	Items []LogEntry `json:"items"`
}

func main() {
	var (
		url       = flag.String("url", "http://localhost:8080/ingest", "Target URL")
		n         = flag.Int("n", 200, "Total requests")
		c         = flag.Int("c", 20, "Concurrency")
		org       = flag.String("org", "acme", "OrgID for generated logs")
		level     = flag.String("level", "info", "Log level")
		apiKey    = flag.String("api-key", "dev-key", "X-API-Key header value")
		timeoutMs = flag.Int("timeout", 10000, "Request timeout in ms")
	)
	flag.Parse()

	client := &http.Client{Timeout: time.Duration(*timeoutMs) * time.Millisecond}

	type result struct {
		ok   bool
		code int
		dur  time.Duration
		err  error
	}

	var (
		wg        sync.WaitGroup
		ch        = make(chan int)
		results   = make([]result, *n)
		startedAt = time.Now()
		doneCount int64
	)

	// workers
	for i := 0; i < *c; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range ch {
				payload := IngestPayload{Items: []LogEntry{
					{OrgID: *org, Level: *level, Message: fmt.Sprintf("loadtest-%d-%d", time.Now().UnixNano(), idx)},
				}}
				body, _ := json.Marshal(payload)
				req, _ := http.NewRequest(http.MethodPost, *url, bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				if apiKey != nil && strings.TrimSpace(*apiKey) != "" {
					req.Header.Set("X-API-Key", *apiKey)
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				dur := time.Since(t0)
				r := result{dur: dur}
				if err != nil {
					r.ok = false
					r.err = err
				} else {
					r.code = resp.StatusCode
					_ = resp.Body.Close()
					r.ok = resp.StatusCode >= 200 && resp.StatusCode < 300
				}
				results[idx] = r
				atomic.AddInt64(&doneCount, 1)
			}
		}()
	}

	// feeder
	go func() {
		for i := 0; i < *n; i++ {
			ch <- i
		}
		close(ch)
	}()

	// wait
	wg.Wait()
	elapsed := time.Since(startedAt)

	// stats
	var okCount, failCount int
	durs := make([]float64, 0, *n)
	var minDur, maxDur time.Duration
	for i := 0; i < *n; i++ {
		r := results[i]
		if r.ok {
			okCount++
		} else {
			failCount++
		}
		durs = append(durs, float64(r.dur)/float64(time.Millisecond))
		if i == 0 || r.dur < minDur {
			minDur = r.dur
		}
		if r.dur > maxDur {
			maxDur = r.dur
		}
	}
	sort.Float64s(durs)
	p := func(x float64) float64 {
		if len(durs) == 0 {
			return 0
		}
		idx := int(math.Ceil(x*float64(len(durs)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(durs) {
			idx = len(durs) - 1
		}
		return durs[idx]
	}

	rps := float64(*n) / (float64(elapsed) / float64(time.Second))
	fmt.Printf("Load test results\n")
	fmt.Printf("Target: %s\n", *url)
	fmt.Printf("Requests: %d  Concurrency: %d  Duration: %.2fs\n", *n, *c, elapsed.Seconds())
	fmt.Printf("Success: %d  Fail: %d  RPS: %.1f\n", okCount, failCount, rps)
	fmt.Printf("Latency ms -> min: %.1f  p50: %.1f  p90: %.1f  p99: %.1f  max: %.1f\n",
		float64(minDur)/float64(time.Millisecond), p(0.50), p(0.90), p(0.99), float64(maxDur)/float64(time.Millisecond))

	if failCount > 0 {
		// exit code 2 to signal partial failures
		os.Exit(2)
	}
}
