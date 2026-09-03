package bundleevents

import (
	"context"
	"testing"
	"time"

	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
)

// notReady is the summary of a bundle whose deployments applied their
// resources, but those are not ready.
func notReady(n int, messages ...string) fleet.BundleSummary {
	s := fleet.BundleSummary{
		DesiredReady: targets,
		Ready:        targets - n,
		NotReady:     n,
	}

	for i, message := range messages {
		s.NonReadyResources = append(s.NonReadyResources, fleet.NonReadyResource{
			Name:    "fleet-default/cluster-" + string(rune('a'+i)),
			State:   fleet.NotReady,
			Message: message,
		})
	}

	return s
}

// waitApplied is the summary of a bundle which was just edited: its targets
// have a new DeploymentID which the agent has not applied yet, so
// GetDeploymentState returns WaitApplied for each of them.
func waitApplied() fleet.BundleSummary {
	return fleet.BundleSummary{DesiredReady: targets, WaitApplied: targets}
}

// settle advances past both the debounce and the minimum interval, and creates
// whatever is due.
func (f *fixture) settle() {
	f.clock.advance(f.opts.MinInterval + f.opts.Debounce + time.Second)
	f.emitter.flush(context.TODO(), *f.opts)
}

// A bundle whose deployments are NotReady is edited to fix it. Its targets move
// to WaitApplied, which is not a failure and not ready either, and only then do
// they become Ready.
func TestEditingAFailingBundleIsNotARecovery(t *testing.T) {
	f := newFixture(t, nil)
	ctx := context.TODO()

	broken := notReady(targets, "deployment app is not ready")

	f.emitter.ObserveBundle(ctx, bundle(ready()), bundle(broken))
	f.settle()

	f.emitter.ObserveBundle(ctx, bundle(broken), bundle(waitApplied()))
	f.settle()

	f.emitter.ObserveBundle(ctx, bundle(waitApplied()), bundle(ready()))
	f.settle()

	got := f.events(t)
	for i, e := range got {
		t.Logf("event %d: %s %s: %q", i+1, e.Type, e.Reason, e.Note)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 events (the failure, then the recovery), got %d", len(got))
	}

	if got[0].Reason != ReasonNotReady {
		t.Errorf("first event: expected reason %s, got %s", ReasonNotReady, got[0].Reason)
	}

	if got[1].Reason != ReasonReady {
		t.Errorf("second event: expected reason %s, got %s", ReasonReady, got[1].Reason)
	}
	if want := "500/500 bundle deployments ready"; got[1].Note != want {
		t.Errorf("second event: expected note %q, got %q", want, got[1].Note)
	}
}
