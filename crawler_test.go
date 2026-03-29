package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

func TestSimpleCrawl(t *testing.T) {
	var mutex sync.Mutex
	visitedPages := make([]string, 0)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				mutex.Lock()
				visitedPages = append(visitedPages, "/")
				mutex.Unlock()
				fmt.Fprint(
					w,
					"<html><body><a href=\"/page2\">link</a></body></html>",
				)
			case "/page2":
				mutex.Lock()
				visitedPages = append(visitedPages, "/page2")
				mutex.Unlock()
				fmt.Fprint(w, "nothing")
			default:
				http.NotFound(w, r)
			}
		}))
	defer server.Close()

	Crawl(server.URL)

	expected := []string{"/", "/page2"}
	if !slices.Equal(expected, visitedPages) {
		t.Errorf("expected to visit pages %v, but got %v", expected, visitedPages)
	}
}
