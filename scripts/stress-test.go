// PodScope Stress Test - Run inside a pod to generate traffic for capture
// Build: go build -o stress-test stress-test.go
// Run:   ./stress-test -c 10 -n 200
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type Endpoint struct {
	Method string
	URL    string
	Body   string
}

// Free APIs - No signup required
var endpoints = []Endpoint{
	// httpbin.org - Echo service
	{Method: "GET", URL: "https://httpbin.org/get"},
	{Method: "POST", URL: "https://httpbin.org/post", Body: `{"test":"data","source":"podscope"}`},
	{Method: "PUT", URL: "https://httpbin.org/put", Body: `{"update":"value"}`},
	{Method: "DELETE", URL: "https://httpbin.org/delete"},
	{Method: "PATCH", URL: "https://httpbin.org/patch", Body: `{"partial":"update"}`},
	{Method: "GET", URL: "https://httpbin.org/headers"},
	{Method: "GET", URL: "https://httpbin.org/ip"},
	{Method: "GET", URL: "https://httpbin.org/uuid"},
	{Method: "GET", URL: "https://httpbin.org/bytes/1024"},
	{Method: "GET", URL: "https://httpbin.org/bytes/4096"},
	{Method: "GET", URL: "https://httpbin.org/bytes/16384"},
	{Method: "GET", URL: "https://httpbin.org/stream/3"},
	{Method: "GET", URL: "https://httpbin.org/delay/1"},
	{Method: "GET", URL: "https://httpbin.org/status/200"},
	{Method: "GET", URL: "https://httpbin.org/status/201"},
	{Method: "GET", URL: "https://httpbin.org/status/204"},
	{Method: "GET", URL: "https://httpbin.org/status/301"},
	{Method: "GET", URL: "https://httpbin.org/status/400"},
	{Method: "GET", URL: "https://httpbin.org/status/404"},
	{Method: "GET", URL: "https://httpbin.org/status/500"},
	{Method: "GET", URL: "https://httpbin.org/status/503"},
	{Method: "GET", URL: "https://httpbin.org/gzip"},
	{Method: "GET", URL: "https://httpbin.org/deflate"},
	{Method: "GET", URL: "https://httpbin.org/html"},
	{Method: "GET", URL: "https://httpbin.org/json"},
	{Method: "GET", URL: "https://httpbin.org/xml"},
	{Method: "GET", URL: "https://httpbin.org/robots.txt"},
	{Method: "GET", URL: "https://httpbin.org/image/png"},
	{Method: "GET", URL: "https://httpbin.org/image/jpeg"},

	// JSONPlaceholder - Fake REST API
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/posts"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/posts/1"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/posts/1/comments"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/comments?postId=1"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/users"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/users/1"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/albums"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/photos?albumId=1"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/todos"},
	{Method: "GET", URL: "https://jsonplaceholder.typicode.com/todos/1"},
	{Method: "POST", URL: "https://jsonplaceholder.typicode.com/posts", Body: `{"title":"stress test","body":"content","userId":1}`},
	{Method: "PUT", URL: "https://jsonplaceholder.typicode.com/posts/1", Body: `{"id":1,"title":"updated","body":"new content","userId":1}`},
	{Method: "PATCH", URL: "https://jsonplaceholder.typicode.com/posts/1", Body: `{"title":"patched"}`},
	{Method: "DELETE", URL: "https://jsonplaceholder.typicode.com/posts/1"},

	// Reqres.in - User API
	{Method: "GET", URL: "https://reqres.in/api/users?page=1"},
	{Method: "GET", URL: "https://reqres.in/api/users?page=2"},
	{Method: "GET", URL: "https://reqres.in/api/users/1"},
	{Method: "GET", URL: "https://reqres.in/api/users/2"},
	{Method: "GET", URL: "https://reqres.in/api/unknown"},
	{Method: "POST", URL: "https://reqres.in/api/users", Body: `{"name":"morpheus","job":"leader"}`},
	{Method: "PUT", URL: "https://reqres.in/api/users/2", Body: `{"name":"neo","job":"the one"}`},
	{Method: "DELETE", URL: "https://reqres.in/api/users/2"},
	{Method: "POST", URL: "https://reqres.in/api/login", Body: `{"email":"eve.holt@reqres.in","password":"cityslicka"}`},

	// Fun APIs
	{Method: "GET", URL: "https://dog.ceo/api/breeds/list/all"},
	{Method: "GET", URL: "https://dog.ceo/api/breeds/image/random"},
	{Method: "GET", URL: "https://dog.ceo/api/breed/hound/images/random"},
	{Method: "GET", URL: "https://catfact.ninja/fact"},
	{Method: "GET", URL: "https://catfact.ninja/facts?limit=5"},

	// World Time API
	{Method: "GET", URL: "https://worldtimeapi.org/api/ip"},
	{Method: "GET", URL: "https://worldtimeapi.org/api/timezone"},
	{Method: "GET", URL: "https://worldtimeapi.org/api/timezone/America/New_York"},
	{Method: "GET", URL: "https://worldtimeapi.org/api/timezone/Europe/London"},

	// Other APIs
	{Method: "GET", URL: "https://api.publicapis.org/entries?category=animals"},
	{Method: "GET", URL: "https://api.publicapis.org/random"},
	{Method: "GET", URL: "https://api.ipify.org?format=json"},
	{Method: "GET", URL: "https://api.genderize.io?name=peter"},
	{Method: "GET", URL: "https://api.nationalize.io?name=michael"},
	{Method: "GET", URL: "https://api.agify.io?name=bella"},

	// DummyJSON - Another fake API
	{Method: "GET", URL: "https://dummyjson.com/products"},
	{Method: "GET", URL: "https://dummyjson.com/products/1"},
	{Method: "GET", URL: "https://dummyjson.com/products/search?q=phone"},
	{Method: "GET", URL: "https://dummyjson.com/users"},
	{Method: "GET", URL: "https://dummyjson.com/users/1"},
	{Method: "GET", URL: "https://dummyjson.com/carts"},
	{Method: "GET", URL: "https://dummyjson.com/quotes"},
	{Method: "GET", URL: "https://dummyjson.com/quotes/random"},
	{Method: "POST", URL: "https://dummyjson.com/products/add", Body: `{"title":"Stress Test Product","price":99}`},
}

type Stats struct {
	success   int64
	failed    int64
	totalTime int64 // nanoseconds
}

func (s *Stats) addSuccess(duration time.Duration) {
	atomic.AddInt64(&s.success, 1)
	atomic.AddInt64(&s.totalTime, int64(duration))
}

func (s *Stats) addFailed() {
	atomic.AddInt64(&s.failed, 1)
}

func makeRequest(ctx context.Context, client *http.Client, ep Endpoint, stats *Stats, verbose bool) {
	start := time.Now()

	var body io.Reader
	if ep.Body != "" {
		body = bytes.NewBufferString(ep.Body)
	}

	req, err := http.NewRequestWithContext(ctx, ep.Method, ep.URL, body)
	if err != nil {
		stats.addFailed()
		if verbose {
			fmt.Printf("✗ %s %s -> error creating request: %v\n", ep.Method, ep.URL, err)
		}
		return
	}

	if ep.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "PodScope-StressTest/1.0")

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		stats.addFailed()
		if verbose {
			fmt.Printf("✗ %s %s -> error: %v\n", ep.Method, ep.URL, err)
		}
		return
	}
	defer resp.Body.Close()

	// Drain body to ensure connection can be reused
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		stats.addSuccess(duration)
		if verbose {
			fmt.Printf("✓ %s %s -> %d (%dms)\n", ep.Method, ep.URL, resp.StatusCode, duration.Milliseconds())
		}
	} else {
		stats.addFailed()
		if verbose {
			fmt.Printf("✗ %s %s -> %d (%dms)\n", ep.Method, ep.URL, resp.StatusCode, duration.Milliseconds())
		}
	}
}

func main() {
	concurrency := flag.Int("c", 10, "Number of concurrent workers")
	totalRequests := flag.Int("n", 100, "Total number of requests")
	timeout := flag.Duration("t", 30*time.Second, "Request timeout")
	verbose := flag.Bool("v", false, "Verbose output (show each request)")
	listEndpoints := flag.Bool("list", false, "List all endpoints and exit")
	noKeepAlive := flag.Bool("no-keepalive", false, "Disable connection reuse (creates more TCP flows)")
	flag.Parse()

	if *listEndpoints {
		fmt.Printf("Available endpoints (%d total):\n\n", len(endpoints))
		for _, ep := range endpoints {
			if ep.Body != "" {
				fmt.Printf("  %s %s\n    Body: %s\n", ep.Method, ep.URL, ep.Body)
			} else {
				fmt.Printf("  %s %s\n", ep.Method, ep.URL)
			}
		}
		return
	}

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              PodScope Stress Test (Go)                       ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Concurrency: %-5d | Requests: %-6d | Timeout: %-4s      ║\n",
		*concurrency, *totalRequests, *timeout)
	keepAliveStatus := "enabled (fewer TCP flows)"
	if *noKeepAlive {
		keepAliveStatus = "DISABLED (1 flow per request)"
	}
	fmt.Printf("║  Endpoints: %-5d | Keep-Alive: %-25s  ║\n", len(endpoints), keepAliveStatus)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	transport := &http.Transport{
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency,
		IdleConnTimeout:     90 * time.Second,
	}

	// Disable keep-alive to create a new TCP connection per request
	if *noKeepAlive {
		transport.DisableKeepAlives = true
		transport.MaxIdleConnsPerHost = -1
	}

	client := &http.Client{
		Timeout:   *timeout,
		Transport: transport,
	}

	stats := &Stats{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create work channel
	work := make(chan Endpoint, *concurrency*2)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ep := range work {
				makeRequest(ctx, client, ep, stats, *verbose)
			}
		}()
	}

	// Feed work
	startTime := time.Now()
	for i := 0; i < *totalRequests; i++ {
		ep := endpoints[rand.Intn(len(endpoints))]
		work <- ep

		// Progress indicator (every 10%)
		if !*verbose && i > 0 && i%(*totalRequests/10) == 0 {
			pct := i * 100 / *totalRequests
			fmt.Printf("\rProgress: %d%% (%d/%d)", pct, i, *totalRequests)
		}
	}
	close(work)

	// Wait for completion
	wg.Wait()
	duration := time.Since(startTime)

	if !*verbose {
		fmt.Printf("\rProgress: 100%% (%d/%d)\n", *totalRequests, *totalRequests)
	}

	// Print results
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      Results Summary                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  ✓ Successful: %-6d                                         ║\n", stats.success)
	fmt.Printf("║  ✗ Failed:     %-6d                                         ║\n", stats.failed)
	fmt.Printf("║  Total:        %-6d                                         ║\n", stats.success+stats.failed)
	fmt.Printf("║  Duration:     %-6.2fs                                        ║\n", duration.Seconds())
	fmt.Printf("║  Throughput:   %-6.1f req/s                                   ║\n",
		float64(stats.success+stats.failed)/duration.Seconds())
	if stats.success > 0 {
		avgMs := float64(stats.totalTime) / float64(stats.success) / float64(time.Millisecond)
		fmt.Printf("║  Avg Latency:  %-6.1f ms                                      ║\n", avgMs)
	}
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	// Output JSON summary for programmatic use
	if os.Getenv("JSON_OUTPUT") == "1" {
		summary := map[string]interface{}{
			"success":      stats.success,
			"failed":       stats.failed,
			"duration_sec": duration.Seconds(),
			"rps":          float64(stats.success+stats.failed) / duration.Seconds(),
		}
		json.NewEncoder(os.Stdout).Encode(summary)
	}
}
