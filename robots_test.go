package main

import (
	"reflect"
	"strings"
	"testing"
)

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
}

func TestRobotsTxtParse(t *testing.T) {
	for _, test := range rulesCases {
		t.Run(test.name, func(t *testing.T) {
			r, err := parseRobotsTxt(strings.TrimSpace(test.input))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			aRules, ok := r.agents["agent"]
			if !ok {
				t.Fatalf("expected to have agent \"agent\": %v", r.agents)
			}

			if !reflect.DeepEqual(aRules.rules, test.expected) {
				t.Errorf("output mismatch: expected %v but got %v", test.expected, r)
			}
		})
	}
}
