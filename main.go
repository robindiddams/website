package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	_ "embed"
)

type HelloResponse struct {
	IP            string `json:"ip"`
	TotalVisitors int64  `json:"total_visitors"`
}

//go:embed index.html
var indexTemplateStr string

var indexTemplate = template.Must(template.New("index").Parse(indexTemplateStr))

func (s *server) helloHandler(w http.ResponseWriter, r *http.Request) {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}

	// For IPv6, RemoteAddr might include the port, so we split it
	if strings.Contains(ip, ":") {
		parts := strings.Split(ip, ":")
		// Check if it's an IPv6 address (e.g., [::1]:12345)
		if len(parts) > 2 && strings.HasPrefix(parts[0], "[") && strings.HasSuffix(parts[len(parts)-2], "]") {
			ip = strings.Join(parts[:len(parts)-1], ":")
		} else {
			ip = parts[0]
		}
	}

	s.totalVisitors.Add(1)

	response := HelloResponse{
		IP:            ip,
		TotalVisitors: s.totalVisitors.Load(),
	}

	err := indexTemplate.Execute(w, response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) sseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	s.activeVisitors.Add(1)
	defer s.activeVisitors.Add(-1)

	lastRead := s.activeVisitors.Load()
	fmt.Fprintf(w, "data: %d\n\n", lastRead)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			fmt.Println("Context done")
			return
		default:
		}
		nextRead := s.activeVisitors.Load()
		if nextRead != lastRead {
			fmt.Fprintf(w, "data: %d\n\n", nextRead)
			lastRead = nextRead
			flusher.Flush()
		}
		time.Sleep(300 * time.Millisecond)
	}
}

type server struct {
	activeVisitors atomic.Int64
	totalVisitors  atomic.Int64
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	s := &server{}
	http.HandleFunc("/", s.helloHandler)
	http.HandleFunc("/sse", s.sseHandler)
	log.Println("Server starting on port " + port + "...")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
