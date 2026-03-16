package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

func filterOutHopByHopHeaders(header http.Header) {
	var hopByHops []string

	// Remove hop-by-hop headers as per RFC 2616 section 13.5.1
	fields := header.Get("Connection")
	if fields != "" {

		hopByHops = strings.Split(fields, ",")
		for i := range hopByHops {
			hopByHops[i] = strings.TrimSpace(hopByHops[i])
		}
	}
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Transfer-Encoding",
		"TE",
		"Trailer",
		"Upgrade",
		"Proxy-Authenticate",
		"Proxy-Authorization",
	}
	hopByHopHeaders = append(hopByHopHeaders, hopByHops...)
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		for key, values := range r.Header {
			log.Printf("%s: %v", key, values)
		}

		// remove hopbyhop headeers from request
		filterOutHopByHopHeaders(r.Header)

		for key, values := range r.Header {
			log.Printf("Tras filtro ---> %s: %v", key, values)
		}

		// hey backend, you there?
		backendAddr := os.Getenv("BACKEND")
		if backendAddr == "" {
			backendAddr = "localhost:8500"
		}
		backendConn, err := net.Dial("tcp", backendAddr)
		if err != nil {
			log.Printf("Error connecting to backend: %v", err)
			http.Error(w, "backend unavailable", http.StatusBadGateway)
			return
		}
		defer backendConn.Close()

		// forward client request to backend
		r.Write(backendConn)

		// reading Backend response
		backendResp, err := http.ReadResponse(
			bufio.NewReader(backendConn),
			r,
		)
		if err != nil {
			http.Error(w, "bad response", http.StatusBadGateway)
			return
		}
		defer backendResp.Body.Close()

		// filter and Copy headers and status
		filterOutHopByHopHeaders(backendResp.Header)
		for key, values := range backendResp.Header {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		w.WriteHeader(backendResp.StatusCode)

		// respond to client
		io.Copy(w, backendResp.Body)
	})

	port := os.Getenv("PROXY_PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("🐐 Goat-Proxy listening on %s 🐐", addr)
	http.ListenAndServe(addr, nil)
}
