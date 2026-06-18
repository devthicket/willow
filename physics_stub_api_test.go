//go:build !physics

package willow_test

import (
	"strings"
	"testing"

	"github.com/devthicket/willow"
)

// In the default build (no -tags physics) the physics package is the stub:
// the public surface still compiles, but the user-facing entry points panic
// with guidance to rebuild with -tags physics, while the per-frame and
// cleanup hooks the engine calls unconditionally stay safe no-ops.

const wantGuidance = "-tags physics"

func wantsPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("%s did not panic in the default (no-physics) build", name)
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("%s panicked with non-string %T: %v", name, r, r)
		}
		if !strings.Contains(msg, wantGuidance) {
			t.Fatalf("%s panic message %q does not mention %q", name, msg, wantGuidance)
		}
	}()
	fn()
}

// TestStub_UserFacingVerbsPanic asserts every "start using physics" entry
// point fails loud in a default build with the rebuild-with -tags physics
// guidance, so a missing physics build is immediately traceable.
func TestStub_UserFacingVerbsPanic(t *testing.T) {
	wantsPanic(t, "EnablePhysics", func() {
		willow.NewContainer("n").EnablePhysics(willow.PhysicsConfig{})
	})
	wantsPanic(t, "SetBody", func() {
		willow.NewContainer("n").SetBody(willow.PhysicsDynamic{})
	})
	wantsPanic(t, "SetBodyEnabled", func() {
		willow.NewContainer("n").SetBodyEnabled(true)
	})
	wantsPanic(t, "StepPhysics", func() {
		willow.NewContainer("n").StepPhysics(1.0 / 60.0)
	})
}

// TestStub_InternalHooksAreNoOps asserts the cleanup/query/per-frame hooks
// the engine invokes regardless of physics usage stay safe no-ops, so a
// default build that never opts into physics runs without panicking.
func TestStub_InternalHooksAreNoOps(t *testing.T) {
	n := willow.NewContainer("n")
	child := willow.NewContainer("child")
	n.AddChild(child) // exercises markPhysicsListDirty internally

	// These must not panic.
	n.DisablePhysics()
	n.RemoveBody()
	n.TickPhysicsTree(1.0 / 60.0)

	if b := n.GetBody(); b != nil {
		t.Fatalf("GetBody in default build = %v, want nil", b)
	}
	if n.BodyEnabled() {
		t.Fatal("BodyEnabled in default build = true, want false")
	}

	// Disposing a subtree that never used physics must be safe too.
	n.Dispose()
}
