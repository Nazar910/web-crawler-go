package main

import "testing"

func TestSimpleRobotsTxt(t *testing.T) {
	r, err := parseRobotsTxt(
		`User-Agent: agent 
Allow: /index`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if r.agents["agent"] == nil {
		t.Fatalf("expected to have agent")
	}

	rules := r.agents["agent"].rules

	if len(rules) != 1 {
		t.Errorf("expected rules to be length of 1 but got %d", len(rules))
	}

	if rules[0].path != "/index" {
		t.Errorf("expected /index but got %s", rules[0].path)
	}

	if !rules[0].allowed {
		t.Errorf("expected rule[0] to be allowed")
	}
}
