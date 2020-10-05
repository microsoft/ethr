package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ethrJSON struct {
	Time                 time.Time `json:"Time"`
	Type                 string    `json:"Type"`
	RemoteAddr           string    `json:"RemoteAddr"`
	Message              string    `json:"Message"`
	Protocol             string    `json:"Protocol"`
	BitsPerSecond        string    `json:"BitsPerSecond"`
	ConnectionsPerSecond string    `json:"ConnectionsPerSecond"`
	PacketsPerSecond     string    `json:"PacketsPerSecond"`
	AverageLatency       string    `json:"AverageLatency"`
}

func serveHTTP(port, ep string, registry *prometheus.Registry) {
	http.Handle(ep, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalln(err)
	}
}

func main() {
	path := flag.String("log", "ethr.log", "path of the log file")
	flag.Parse()
	c := newCollector(*path)
	registry := prometheus.NewRegistry()
	if err := registry.Register(c); err != nil {
		log.Println(err)
	}
	serveHTTP(":8093", "/metrics", registry)
}
