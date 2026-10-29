package world

import (
	"errors"
	"testing"

	"github.com/JotaEspig/autonomous-agent-simulator/internal/model"
)

func newTestWorld(t *testing.T) (*World, AgentID, AgentID) {
	t.Helper()
	m, err := model.Load("testdata/tx")
	if err != nil {
		t.Fatal(err)
	}
	w := New(m)
	if err := w.Populate(1); err != nil {
		t.Fatal(err)
	}
	alice, _ := w.Find("alice")
	bob, _ := w.Find("bob")
	return w, alice, bob
}

func attr(w *World, id AgentID, name string) model.Value {
	f := w.TypeOf(id).AttrIndex[name]
	v, _ := w.Get(id, f)
	return v
}

func self(w *World, id AgentID, name string) Ref {
	return Ref{Field: w.TypeOf(id).AttrIndex[name], Agent: id}
}

func ctx(w *World, name string) Ref { return Ref{Field: w.Model.ContextIndex[name]} }

func TestTransferIsAtomicAndConserving(t *testing.T) {
	w, alice, _ := newTestWorld(t)
	actor := Actor{ID: alice}
	reason, err := w.Apply(actor, []*Effect{{Op: OpTransfer, Source: self(w, alice, "cash"), Target: ctx(w, "pot"), Value: model.Int(30)}})
	if err != nil || reason != "" {
		t.Fatalf("transfer failed: %q %v", reason, err)
	}
	if got := attr(w, alice, "cash").I + w.Context.Get(w.Model.ContextIndex["pot"]).I; got != 100 {
		t.Fatalf("money not conserved: %d", got)
	}

	// Second effect violates min: 0, so the first one must be rolled back too.
	reason, err = w.Apply(actor, []*Effect{
		{Op: OpAdd, Target: ctx(w, "pot"), Value: model.Int(5)},
		{Op: OpAdd, Target: self(w, alice, "cash"), Value: model.Int(-1000)},
	})
	if err != nil || reason == "" {
		t.Fatalf("expected a rejection, got %q %v", reason, err)
	}
	if pot := w.Context.Get(w.Model.ContextIndex["pot"]).I; pot != 30 {
		t.Fatalf("rollback failed: pot = %d", pot)
	}
	if cash := attr(w, alice, "cash").I; cash != 70 {
		t.Fatalf("rollback failed: cash = %d", cash)
	}
}

func TestClampInsteadOfReject(t *testing.T) {
	w, alice, _ := newTestWorld(t)
	reason, err := w.Apply(Actor{ID: alice}, []*Effect{{Op: OpAdd, Target: self(w, alice, "mood"), Value: model.Float(5)}})
	if err != nil || reason != "" {
		t.Fatalf("unexpected rejection: %q %v", reason, err)
	}
	if mood := attr(w, alice, "mood").F; mood != 1 {
		t.Fatalf("mood = %v, want clamped to 1", mood)
	}
}

func TestAgentsCannotTouchOthers(t *testing.T) {
	w, alice, bob := newTestWorld(t)
	_, err := w.Apply(Actor{ID: alice}, []*Effect{{Op: OpAdd, Target: self(w, bob, "cash"), Value: model.Int(1)}})
	var me *ModelError
	if !errors.As(err, &me) {
		t.Fatalf("expected a model error, got %v", err)
	}
	rule := &model.Rule{Name: "r"}
	if reason, err := w.Apply(Actor{Rule: rule}, []*Effect{{Op: OpAdd, Target: self(w, bob, "cash"), Value: model.Int(1)}}); err != nil || reason != "" {
		t.Fatalf("rules may change any agent: %q %v", reason, err)
	}
}

func TestMergePolicies(t *testing.T) {
	w, alice, bob := newTestWorld(t)
	w.BeginPhase()
	set := func(id AgentID, field string) (string, error) {
		return w.Apply(Actor{ID: id}, []*Effect{{Op: OpSet, Target: ctx(w, field), Value: model.String("x")}})
	}
	if _, err := set(alice, "flag"); err != nil {
		t.Fatal(err)
	}
	if _, err := set(bob, "flag"); err == nil {
		t.Fatal("two writers of a merge:error field must stop the run")
	}
	if _, err := set(alice, "first"); err != nil {
		t.Fatal(err)
	}
	if reason, err := set(bob, "first"); err != nil || reason == "" {
		t.Fatalf("merge:first must reject the second writer, got %q %v", reason, err)
	}
	w.BeginPhase()
	if reason, err := set(bob, "flag"); err != nil || reason != "" {
		t.Fatalf("a new phase starts a new write set: %q %v", reason, err)
	}
}

func TestSpawnLinkRemove(t *testing.T) {
	w, alice, _ := newTestWorld(t)
	typ := w.TypeOf(alice)
	spawn := &Effect{Op: OpSpawn, Type: typ, Attrs: []AttrValue{{Field: typ.AttrIndex["cash"], Value: model.Int(5)}}}
	effs := []*Effect{spawn, {Op: OpLink, Relation: w.Model.RelationIndex["friend"], Agent: Endpoint{ID: alice}, Other: Endpoint{Spawn: spawn}}}
	if reason, err := w.Apply(Actor{ID: alice}, effs); err != nil || reason != "" {
		t.Fatalf("spawn failed: %q %v", reason, err)
	}
	child := spawn.Spawned()
	if attr(w, child, "cash").I != 5 || len(w.Graphs[0].Out(alice)) != 1 {
		t.Fatal("spawned agent not initialized or not linked")
	}
	if reason, err := w.Apply(Actor{ID: child}, []*Effect{{Op: OpRemove, Agent: Endpoint{ID: child}}}); err != nil || reason != "" {
		t.Fatalf("remove failed: %q %v", reason, err)
	}
	if w.Exists(child) || len(w.Graphs[0].Out(alice)) != 0 || w.Population() != 2 {
		t.Fatal("removed agent still referenced")
	}
	if attr(w, alice, "cash").I != 100 {
		t.Fatal("swap-remove corrupted another agent")
	}
}

func TestMemoryRing(t *testing.T) {
	w, alice, _ := newTestWorld(t)
	for i := 1; i <= 5; i++ {
		if _, err := w.Apply(Actor{ID: alice}, []*Effect{{Op: OpRemember, Memory: "seen", Value: model.Int(int64(i)), Agent: Endpoint{ID: alice}}}); err != nil {
			t.Fatal(err)
		}
	}
	l, _ := w.Locate(alice)
	st := w.Stores[l.Type]
	mem := st.Type.MemoryIndex["seen"]
	if st.MemLen(int(l.Index), mem) != 3 {
		t.Fatal("memory must be bounded by its capacity")
	}
	for back, want := range []int64{5, 4, 3} {
		if v, _ := st.Recall(int(l.Index), mem, back); v.I != want {
			t.Fatalf("recall(%d) = %d, want %d", back, v.I, want)
		}
	}
}
