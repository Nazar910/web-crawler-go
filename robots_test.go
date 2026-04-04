package main

import "testing"

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
