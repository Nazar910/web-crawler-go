package main

import (
	"fmt"
	"net/http"
	"net/url"
)

type robotsTxt interface {
	IsPathAllowed(path string) bool
}

type noop struct{}

func (r *noop) IsPathAllowed(path string) bool { return true }

// this will actually be checking robots.txt
type robotsChecker struct {
	agent  string
	rRules robotsRules
}

// checks whether particular path is allowed for this agent
// TODO: naive implementation, replace with longest path match check
// TODO: consider adding * support
func (r *robotsChecker) IsPathAllowed(path string) bool {
	url, err := url.Parse(path)

	if err != nil {
		return false
	}

	clearedPath := url.Path

	if clearedPath == "" {
		// it should be the root path then
		// set it to /
		clearedPath = "/"
	}

	var allowed bool

	if starAgent, ok := r.rRules.agents["*"]; ok {
		for _, r := range starAgent.rules {
			if r.path == clearedPath {
				allowed = r.allowed
			}
		}
	}

	if agent, ok := r.rRules.agents[r.agent]; ok {
		for _, r := range agent.rules {
			if r.path == clearedPath {
				allowed = r.allowed
			}
		}
	}

	return allowed
}

type rule struct {
	path    string
	allowed bool
}

type agentRules struct {
	rules []rule
}

type robotsRules struct {
	agents map[string]*agentRules
}

func (r *robotsRules) allowPath(agent, path string) {
	if _, ok := r.agents[agent]; !ok {
		r.agents[agent] = &agentRules{
			rules: make([]rule, 0),
		}
	}

	aRules := r.agents[agent]
	for i := range aRules.rules {
		if aRules.rules[i].path == path {
			aRules.rules[i].allowed = true
			return
		}
	}

	r.agents[agent].rules = append(r.agents[agent].rules, rule{path, true})
}

func (r *robotsRules) disallowPath(agent, path string) {
	if _, ok := r.agents[agent]; !ok {
		r.agents[agent] = &agentRules{
			rules: make([]rule, 0),
		}
	}

	aRules := r.agents[agent]
	for i := range aRules.rules {
		if aRules.rules[i].path == path {
			aRules.rules[i].allowed = false
			return
		}
	}

	r.agents[agent].rules = append(r.agents[agent].rules, rule{path, false})
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

	parsed, err := parseRobotsTxt(res.Body)

	if err != nil {
		return nil, err
	}

	return &robotsChecker{agent, parsed}, nil
}
