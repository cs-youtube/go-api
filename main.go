// A small REST API, written the idiomatic way with the standard library.
// cs-youtube/rust-api has the same endpoints and metrics, so the two can be compared under load.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Item struct {
	ID     uint64   `json:"id"`
	Name   string   `json:"name"`
	Price  float64  `json:"price"`
	Active bool     `json:"active"`
	Tags   []string `json:"tags"`
}

// items is the model: created once at start and only read after that.
var items = newItems(1000)

func newItems(count int) []Item {
	list := make([]Item, count)
	for i := range list {
		id := uint64(i)
		list[i] = Item{
			ID:     id,
			Name:   "item-" + strconv.FormatUint(id, 10),
			Price:  float64(id) * 1.5,
			Active: id%2 == 0,
			Tags:   []string{"alpha", "beta", "gamma"},
		}
	}
	return list
}

func intParam(r *http.Request, name string, fallback, max int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get(name)); err == nil && v >= 0 && v <= max {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Print(err)
	}
}

func listItems(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, items[:intParam(r, "limit", 50, len(items))])
}

func getItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if id >= uint64(len(items)) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, items[id])
}

func createItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil || item.Name == "" {
		http.Error(w, "bad item", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// statusWriter remembers the status code for the metrics.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func newHandler(registry *prometheus.Registry) http.Handler {
	requests := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Time from receiving a request to finishing its response.",
		Buckets: []float64{.000025, .00005, .0001, .00025, .0005, .001, .002, .005, .01, .02, .05, .1, .2, .5, 1, 2, 5, 10},
	}, []string{"method", "route", "code"})
	inFlight := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "Requests being handled right now.",
	})
	registry.MustRegister(requests, inFlight)

	mux := http.NewServeMux()
	handle := func(method, route string, h http.HandlerFunc) {
		mux.HandleFunc(method+" "+route, func(w http.ResponseWriter, r *http.Request) {
			inFlight.Inc()
			defer inFlight.Dec()
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
			h(sw, r)
			requests.WithLabelValues(method, route, strconv.Itoa(sw.code)).Observe(time.Since(start).Seconds())
		})
	}
	handle("GET", "/items", listItems)
	handle("POST", "/items", createItem)
	handle("GET", "/items/{id}", getItem)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	mux.Handle("GET /metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return mux
}

func main() {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	log.Fatal(http.ListenAndServe(":8080", newHandler(registry)))
}
