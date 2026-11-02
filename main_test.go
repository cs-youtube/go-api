package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func request(method, url, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	newHandler(prometheus.NewRegistry()).ServeHTTP(rec, httptest.NewRequest(method, url, strings.NewReader(body)))
	return rec
}

func TestLatencyHistogramHasTheAgreedBuckets(t *testing.T) {
	registry := prometheus.NewRegistry()
	h := newHandler(registry)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/items/1", nil))
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var got []float64
	for _, f := range families {
		if f.GetName() == "http_request_duration_seconds" {
			for _, b := range f.GetMetric()[0].GetHistogram().GetBucket() {
				got = append(got, b.GetUpperBound())
			}
		}
	}
	// Must equal the Rust service's list (cs-youtube/rust-api).
	want := []float64{.000025, .00005, .0001, .00025, .0005, .001, .002, .005, .01, .02, .05, .1, .2, .5, 1, 2, 5, 10}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListItemsHonoursLimit(t *testing.T) {
	var list []Item
	rec := request("GET", "/items?limit=3", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 3 {
		t.Fatalf("got %s (%v)", rec.Body, err)
	}
}

func TestGetItem(t *testing.T) {
	var item Item
	rec := request("GET", "/items/7", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil || item.Name != "item-7" {
		t.Fatalf("got %s (%v)", rec.Body, err)
	}
	if rec := request("GET", "/items/x", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id: got %d", rec.Code)
	}
	if rec := request("GET", "/items/1000", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id: got %d", rec.Code)
	}
}

func TestCreateItem(t *testing.T) {
	if rec := request("POST", "/items", `{"name":"a"}`); rec.Code != http.StatusCreated {
		t.Fatalf("got %d", rec.Code)
	}
	if rec := request("POST", "/items", `{"name":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: got %d", rec.Code)
	}
}
