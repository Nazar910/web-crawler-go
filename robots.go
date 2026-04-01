package main

import (
	"fmt"
	"io"
	"net/http"
)

type robotsTxt interface {
	IsUserAgentAllowed() bool
	IsLinkAllowed(link string) bool
}

type noop struct{}

func (r *noop) IsUserAgentAllowed() bool       { return true }
func (r *noop) IsLinkAllowed(link string) bool { return true }

// this will actually be checking robots.txt
type robotsChecker struct {
	agent string
}

func (r *robotsChecker) IsUserAgentAllowed() bool       { return true }
func (r *robotsChecker) IsLinkAllowed(link string) bool { return true }

func parseRobotsTxt(agent, input string) *robotsChecker {
	return &robotsChecker{agent}
}

// fetches robots.txt from the specified domain
// if not found -> uses noop robots check which basically allows everything
// otherwise, it will parse robots.txt and return a proper robots checker
func newRobots(domain, agent string) (robotsTxt, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/robots.txt", domain), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", agent)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return &noop{}, nil
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	robots := parseRobotsTxt(agent, string(b))
	return robots, nil
}
