package main

import (
	"slices"
	"strings"
	"testing"
)

func dedent(s string) string {
	var sb strings.Builder
	for l := range strings.SplitSeq(s, "\n") {
		sb.WriteString(strings.TrimLeft(l, "\t"))
		sb.WriteString("\n")
	}
	return sb.String()
}

func TestRobotsTxtParse(t *testing.T) {
	var rulesCases = []struct {
		name     string
		input    string
		expected []rule
	}{
		{
			name: "simple allow",
			input: `
		User-agent: agent
		Allow: /index`,
			expected: []rule{{"/index", true}},
		},
		{
			name: "simple disallow",
			input: `
		User-agent: agent
		Disallow: /index`,
			expected: []rule{{"/index", false}},
		},
		{
			name: "allow and disallow",
			input: `
		User-agent: agent
		Allow: /
		Disallow: /internal`,
			expected: []rule{{"/", true}, {"/internal", false}},
		},
		{
			name: "disallow which invalidate allow",
			input: `
		User-agent: agent
		Allow: /
		Disallow: /`,
			expected: []rule{{"/", false}},
		},

		{
			name: "allow which invalidate disallow",
			input: `
		User-agent: agent
		Disallow: /
		Allow: /`,
			expected: []rule{{"/", true}},
		},
	}
	for _, test := range rulesCases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r, err := parseRobotsTxt(strings.TrimSpace(dedent(test.input)))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			aRules, ok := r.agents["agent"]
			if !ok {
				t.Fatalf("expected to have agent \"agent\": %v", r.agents)
			}

			if !slices.Equal(aRules.rules, test.expected) {
				t.Errorf("output mismatch: expected %v but got %v", test.expected, aRules.rules)
			}
		})
	}
}

func TestRobotsMultipleAgents(t *testing.T) {
	input := dedent(`
	User-agent: *
	Disallow: /

	User-agent: agent
	Allow: /
	`)
	r, err := parseRobotsTxt(input)

	if err != nil {
		t.Fatalf("expected no error but got %v", err)
	}

	if len(r.agents) != 2 {
		t.Errorf("expected agents number to be 2 but got %d", len(r.agents))
	}

	all, ok := r.agents["*"]

	if !ok {
		t.Fatal("expected to have entry for *")
	}

	expectedAllRules := []rule{{"/", false}}
	if !slices.Equal(all.rules, expectedAllRules) {
		t.Errorf("all mismatch: expected %v but got %v", expectedAllRules, all.rules)
	}

	agent, ok := r.agents["agent"]

	if !ok {
		t.Fatal("expected to have entry for \"agent\"")
	}

	expectedAgentRules := []rule{{"/", true}}

	if !slices.Equal(agent.rules, expectedAgentRules) {
		t.Errorf("agent rules mismatch: expected %v got %v", expectedAgentRules, agent.rules)
	}
}

func TestRobotsChecker(t *testing.T) {
	rRules := robotsRules{
		agents: map[string]*agentRules{
			"agent": &agentRules{
				rules: []rule{{"/", true}, {"/internal", false}},
			},
		},
	}

	checker := robotsChecker{
		agent:  "agent",
		rRules: rRules,
	}

	if !checker.IsPathAllowed("http://foo.bar") {
		t.Error("expected / route to be allowed")
	}

	if checker.IsPathAllowed("http://foo.bar/internal") {
		t.Error("expected /internal to not be allowed")
	}
}
