package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Config holds all runtime configuration read from environment variables.
type Config struct {
	Backends       []string
	Port           string
	BackendTimeout time.Duration
}

func loadConfig() Config {
	backendsEnv := os.Getenv("BACKENDS")
	if backendsEnv == "" {
		slog.Error("BACKENDS environment variable is required")
		os.Exit(1)
	}
	backends := strings.Split(backendsEnv, ",")
	for i := range backends {
		backends[i] = strings.TrimSpace(backends[i])
	}

	timeoutSecs := 2
	if v := os.Getenv("BACKEND_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSecs = n
		}
	}

	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8080"
	}

	return Config{
		Backends:       backends,
		Port:           port,
		BackendTimeout: time.Duration(timeoutSecs) * time.Second,
	}
}

// LoadBalancer distributes requests across backends using round-robin.
type LoadBalancer struct {
	backends []string
	current  uint64
}

func (lb *LoadBalancer) next() string {
	idx := atomic.AddUint64(&lb.current, 1)
	return lb.backends[idx%uint64(len(lb.backends))]
}

// ReverseProxy forwards incoming HTTP requests to a pool of backends.
type ReverseProxy struct {
	lb      *LoadBalancer
	timeout time.Duration
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	filterOutHopByHopHeaders(r.Header)

	// clientIP comes from the TCP connection — it cannot be spoofed.
	// We explicitly delete any client-supplied X-Forwarded-For and X-Real-IP
	// before setting them, so backends always receive trustworthy values.
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		r.Header.Del("X-Forwarded-For")
		r.Header.Del("X-Real-Ip")
		r.Header.Set("X-Forwarded-For", clientIP)
		r.Header.Set("X-Real-Ip", clientIP)
	}

	backendAddr := p.lb.next()
	backendConn, err := net.DialTimeout("tcp", backendAddr, p.timeout)
	if err != nil {
		slog.Error("backend unavailable", "backend", backendAddr, "error", err,
			"method", r.Method, "path", r.URL.Path)
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer backendConn.Close()

	if err := r.Write(backendConn); err != nil {
		slog.Error("failed to forward request", "backend", backendAddr, "error", err,
			"method", r.Method, "path", r.URL.Path)
		http.Error(w, "failed to forward request", http.StatusBadGateway)
		return
	}

	backendResp, err := http.ReadResponse(bufio.NewReader(backendConn), r)
	if err != nil {
		slog.Error("bad response from backend", "backend", backendAddr, "error", err,
			"method", r.Method, "path", r.URL.Path)
		http.Error(w, "bad response from backend", http.StatusBadGateway)
		return
	}
	defer backendResp.Body.Close()

	filterOutHopByHopHeaders(backendResp.Header)
	for key, values := range backendResp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(backendResp.StatusCode)
	io.Copy(w, backendResp.Body)

	slog.Info("request",
		"method", r.Method,
		"path", r.URL.Path,
		"status", backendResp.StatusCode,
		"backend", backendAddr,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

// filterOutHopByHopHeaders removes hop-by-hop headers as per RFC 2616 §13.5.1.
func filterOutHopByHopHeaders(header http.Header) {
	hopByHops := []string{
		"Connection",
		"Keep-Alive",
		"Transfer-Encoding",
		"TE",
		"Trailer",
		"Upgrade",
		"Proxy-Authenticate",
		"Proxy-Authorization",
	}
	// Also remove any headers listed in the Connection header itself.
	if fields := header.Get("Connection"); fields != "" {
		for _, f := range strings.Split(fields, ",") {
			hopByHops = append(hopByHops, strings.TrimSpace(f))
		}
	}
	for _, h := range hopByHops {
		header.Del(h)
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := loadConfig()

	proxy := &ReverseProxy{
		lb:      &LoadBalancer{backends: cfg.Backends},
		timeout: cfg.BackendTimeout,
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	slog.Info("goat-proxy starting",
		"addr", addr,
		"backends", cfg.Backends,
		"backend_timeout", cfg.BackendTimeout.String(),
	)

	if err := http.ListenAndServe(addr, proxy); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
