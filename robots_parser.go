package main

import (
	"bytes"
	"fmt"
	"io"
)

type parser struct {
	s scanner
	r robotsRules

	token        Token
	currentAgent string
}

func (p *parser) eat(expectedType TokenType) error {
	if p.token.tokenType == expectedType {
		nextToken, err := p.s.NextToken()
		p.token = nextToken
		return err
	}

	return fmt.Errorf("unexpected token: %v (expected %v)", p.token, expectedType)
}

func (p *parser) process() error {
	left := p.token.value
	err := p.eat(String)

	if err != nil {
		return err
	}

	err = p.eat(Colon)

	if err != nil {
		return err
	}

	right := p.token.value
	err = p.eat(String)

	if err != nil {
		return err
	}

	switch leftL := string(bytes.ToLower([]byte(left))); leftL {
	case "user-agent":
		p.currentAgent = right
	case "allow":
		p.r.allowPath(p.currentAgent, right)
	case "disallow":
		p.r.disallowPath(p.currentAgent, right)
	case "sitemap":
		p.r.sitemaps = append(p.r.sitemaps, right)
	default:
		return fmt.Errorf("unknown lhs: %s", leftL)
	}

	return nil
}

func parseRobotsTxt(reader io.Reader) (robotsRules, error) {
	s, err := newScanner(reader)
	if err != nil {
		return robotsRules{}, err
	}
	p := parser{
		s: s,
		r: robotsRules{
			agents: make(map[string]*agentRules),
		},
	}

	token, err := p.s.NextToken()
	p.token = token

	for err == nil && p.token.tokenType != Eof {
		err = p.process()
	}

	return p.r, err
}
