package script

import (
	"strings"
	"testing"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/rng"
	"github.com/JotaEspig/autonomous-agent-simulator/internal/world"
)

func TestSandbox(t *testing.T) {
	m, err := model.Load("testdata/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	c, err := Compile(m)
	if err != nil {
		t.Fatal(err)
	}
	w := world.New(m)
	if err := w.Populate(1); err != nil {
		t.Fatal(err)
	}
	rt, err := c.NewRuntime(w)
	if err != nil {
		t.Fatal(err)
	}
	defer rt.Close()
	src := rng.New()
	call := func(fn string) (Result, error) {
		f := rt.Lookup("s.lua", fn)
		if f == 0 {
			t.Fatalf("function %s not found", fn)
		}
		return rt.CallAgent(f, 1, src, false)
	}
	for fn, want := range map[string]string{
		"uses_random":   "math.random is disabled",
		"writes_global": "must not keep state in globals",
		"reads_hidden":  "is hidden from agents",
		"reads_private": "is private",
		"writes_self":   "cannot be modified directly",
	} {
		if _, err := call(fn); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want error containing %q, got %v", fn, want, err)
		}
	}
	if r, err := call("reads_public"); err != nil || r.Kind != ResultNumber {
		t.Errorf("reads_public: %v %v", r, err)
	}
	if r, err := call("sorted_pairs"); err != nil || !r.Truthy() {
		t.Errorf("pairs must iterate numbers then strings in order: %v %v", r, err)
	}
	// Rules and other privileged callers see everything.
	if _, err := rt.CallAgent(rt.Lookup("s.lua", "reads_hidden"), 1, src, true); err != nil {
		t.Errorf("privileged call: %v", err)
	}
}
