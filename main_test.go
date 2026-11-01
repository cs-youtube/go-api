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

func TestListItemsHonoursLimit(t *testing.T) {
	var items []Item
	rec := request("GET", "/items?limit=3", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil || len(items) != 3 {
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
}

func TestCreateItem(t *testing.T) {
	if rec := request("POST", "/items", `{"name":"a"}`); rec.Code != http.StatusCreated {
		t.Fatalf("got %d", rec.Code)
	}
	if rec := request("POST", "/items", `{"name":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty name: got %d", rec.Code)
	}
}
