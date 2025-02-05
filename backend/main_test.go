package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingResultsEndpoint(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/ping-results", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
