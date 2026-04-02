package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"unicode"
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
	agent  string
	rRules robotsRules
}

func (r *robotsChecker) IsUserAgentAllowed() bool       { return true }
func (r *robotsChecker) IsLinkAllowed(link string) bool { return true }

type parser struct {
	buf    []byte
	pos    int
	parsed map[string][]string
}

func (p *parser) skipSpace() {
	for p.buf[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) skipWs() {
	for unicode.IsSpace(rune(p.buf[p.pos])) {
		p.pos++
	}
}

func (p *parser) userAgent() (string, error) {
	start := p.pos

	for unicode.IsLetter(rune(p.buf[p.pos])) || p.buf[p.pos] == '-' {
		p.pos++
	}

	uagentKw := p.buf[start:p.pos]

	if !bytes.Equal(bytes.ToLower(uagentKw), []byte("user-agent")) {
		return "", fmt.Errorf("cannot parse %s", uagentKw)
	}

	p.skipSpace()

	if p.buf[p.pos] != ':' {
		return "", fmt.Errorf("expected ':', got %b", p.buf[p.pos])
	}

	p.pos++

	p.skipSpace()

	start = p.pos

	for !unicode.IsSpace(rune(p.buf[p.pos])) {
		p.pos++
	}

	agentStr := string(p.buf[start:p.pos])

	p.skipWs()

	return agentStr, nil
}

func (p *parser) allow() (string, error) {
	start := p.pos
	for unicode.IsLetter(rune(p.buf[p.pos])) {
		p.pos++
	}

	allowKw := string(p.buf[start:p.pos])

	if allowKw != "Allow" {
		return "", errors.New("wrong allow keyword")
	}

	p.skipSpace()

	if p.buf[p.pos] != ':' {
		return "", fmt.Errorf("expected ':', got %c", p.buf[p.pos])
	}
	p.pos++

	p.skipSpace()

	start = p.pos
	for p.pos < len(p.buf) && !unicode.IsSpace(rune(p.buf[p.pos])) {
		p.pos++
	}

	path := string(p.buf[start:p.pos])

	return path, nil
}

func (p *parser) disallow() (string, error) {
	start := p.pos

	for unicode.IsLetter(rune(p.buf[p.pos])) {
		p.pos++
	}

	disallowKw := string(p.buf[start:p.pos])

	if disallowKw != "Disallow" {
		return "", errors.New("wrong disallow keyword")
	}

	p.skipSpace()

	if p.buf[p.pos] != ':' {
		return "", fmt.Errorf("expected ':', got %c", p.buf[p.pos])
	}

	p.pos++
	p.skipSpace()

	start = p.pos

	for p.pos < len(p.buf) && !unicode.IsSpace(rune(p.buf[p.pos])) {
		p.pos++
	}

	path := string(p.buf[start:p.pos])

	return path, nil
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

func parseRobotsTxt(input string) (robotsRules, error) {
	p := &parser{[]byte(input), 0, make(map[string][]string)}
	r := robotsRules{
		agents: make(map[string]*agentRules),
	}

	var currAgent string
	for p.pos < len(p.buf) {
		switch p.buf[p.pos] {
		case 'U', 'u':
			var err error
			currAgent, err = p.userAgent()
			if err != nil {
				return robotsRules{}, err
			}
			r.agents[currAgent] = &agentRules{
				rules: make([]rule, 0),
			}
		case 'A':
			path, err := p.allow()
			if err != nil {
				return robotsRules{}, err
			}
			r.agents[currAgent].rules = append(r.agents[currAgent].rules, rule{path, true})
		case 'D':
			path, err := p.disallow()
			if err != nil {
				return robotsRules{}, err
			}

			r.agents[currAgent].rules = append(r.agents[currAgent].rules, rule{path, false})
		default:
			return robotsRules{}, fmt.Errorf("unknown char: %c", p.buf[p.pos])
		}
	}

	return r, nil
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

	parsed, err := parseRobotsTxt(string(b))

	if err != nil {
		return nil, err
	}

	return &robotsChecker{agent, parsed}, nil
}
