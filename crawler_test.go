package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
)

const indexHtml = `
<html>
<body>
	<a href="/page2">link</a>
</body>
</html>
`

type mockServer struct {
	server  *httptest.Server
	visited []string
	paths   map[string]string

	mu sync.Mutex
}

func newMockServer(paths map[string]string) *mockServer {
	mockServer := &mockServer{
		visited: make([]string, 0),
		paths:   paths,
	}

	mockServer.initHttp()

	return mockServer
}

func (c *mockServer) initHttp() {
	c.server = httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.mu.Lock()
			c.visited = append(c.visited, r.URL.Path)
			c.mu.Unlock()

			if resp, ok := c.paths[r.URL.Path]; ok {
				fmt.Fprint(w, resp)
			} else {
				http.NotFound(w, r)
			}
		}))
}

func TestSimpleCrawl(t *testing.T) {
	paths := map[string]string{
		"/":      indexHtml,
		"/page2": "nothing",
	}
	mockServer := newMockServer(paths)
	defer mockServer.server.Close()

	Crawl(mockServer.server.URL)

	expected := []string{"/robots.txt", "/", "/page2"}
	if !slices.Equal(expected, mockServer.visited) {
		t.Errorf("expected to visit pages %v, but got %v", expected, mockServer.visited)
	}
}

func TestDuplicateLinks(t *testing.T) {
	paths := map[string]string{
		"/":      indexHtml,
		"/page2": `<a href="/page3">link to page3</a><a href="/page4">0_0</a>`,
		"/page3": `<a href="/page4">link</a>`,
		"/page4": "nothing",
	}
	mock := newMockServer(paths)
	defer mock.server.Close()

	Crawl(mock.server.URL)

	expected := []string{"/", "/page2", "/page3", "/page4", "/robots.txt"}
	slices.Sort(mock.visited)
	if !slices.Equal(expected, mock.visited) {
		t.Errorf("expected to visit pages %v, but got %v", expected, mock.visited)
	}
}
