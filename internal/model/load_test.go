package model

import (
	"strings"
	"testing"
)

func TestLoadReportsEveryError(t *testing.T) {
	_, err := Load("testdata/broken")
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	for _, want := range []string{
		"simulation.maxTicks: must be > 0",
		`"id" is reserved`,
		`unknown type "flaot"`,
		`unknown decision "missing"`,
		`set exactly one of "agg", "context", "agent" or "script"`,
		`unknown field "atributes"`,
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("error report is missing %q:\n%s", want, msg)
		}
	}
}

func TestLoadExampleModels(t *testing.T) {
	for _, dir := range []string{"../../models/prisoners-dilemma", "../../models/entrepreneurship", "../../models/bread-island", "../../models/benchmark"} {
		if _, err := Load(dir); err != nil {
			t.Errorf("%s: %v", dir, err)
		}
	}
}
