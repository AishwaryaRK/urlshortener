package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type MockUrlShortener struct {
	urlMap         sync.Map
	invertedUrlMap sync.Map
	hostname       string
	len            int
}

func (m *MockUrlShortener) loadFromInvertedUrlMap(s string) (any, bool) {
	return "https://www.cs.cmu.edu/afs/cs.cmu.edu/project/phrensy/pub/papers/DilleyMPPSW02.pdf", true
}

func (m *MockUrlShortener) loadFromUrlMap(originalUrl string) (any, bool) {
	return "", false
}

func (m *MockUrlShortener) isValidHostname(host string) bool {
	return true
}

func (m *MockUrlShortener) createShortenedUrl(originalUrl string) (string, error) {
	return "https://urlshortener.com/aBcd1", nil
}

func TestGetShortenedUrl(t *testing.T) {
	req := httptest.NewRequest("GET", "/shortenurl", strings.NewReader(`{"url":"https://www.cs.cmu.edu/afs/cs.cmu.edu/project/phrensy/pub/papers/DilleyMPPSW02.pdf"}`))
	rr := httptest.NewRecorder()
	mockUrlShortener := &MockUrlShortener{hostname: "urlshortener.com", len: 5}
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		getShortenedUrl(writer, request, mockUrlShortener)
	})
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected http response status to be 201 StatusCreated but got %v", rr.Code)
	}

	expectedBody := map[string]string{"shortenedUrl": "https://urlshortener.com/aBcd1"}
	var actualBody map[string]string
	err := json.Unmarshal(rr.Body.Bytes(), &actualBody)
	if err != nil {
		t.Fatalf("expected no err in unmarshalling response body, got %v", err)
	}
	if !reflect.DeepEqual(actualBody, expectedBody) {
		t.Errorf("expected response: %v, actual: %v", expectedBody, actualBody)
	}
}
