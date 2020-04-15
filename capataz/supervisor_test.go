package capataz_test

//
// NOTE: If you feel it is counter-intuitive to have workers start before
// supervisors in the assertions bellow, check stest/README.md
//

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/capatazlib/go-capataz/capataz"
	. "github.com/capatazlib/go-capataz/internal/stest"
)

func TestStartSingleChild(t *testing.T) {
	events, err := ObserveSupervisor(
		context.TODO(),
		"root",
		capataz.WithNodes(WaitDoneWorker("one")),
		[]capataz.Opt{},
		func(EventManager) {},
	)

	assert.NoError(t, err)
	AssertExactMatch(t, events,
		[]EventP{
			WorkerStarted("root/one"),
			SupervisorStarted("root"),
			WorkerTerminated("root/one"),
			SupervisorTerminated("root"),
		})
}

// Test a supervision tree with three children start and stop in the default
// order (LeftToRight)
func TestStartMutlipleChildrenLeftToRight(t *testing.T) {
	events, err := ObserveSupervisor(
		context.TODO(),
		"root",
		capataz.WithNodes(
			WaitDoneWorker("child0"),
			WaitDoneWorker("child1"),
			WaitDoneWorker("child2"),
		),
		[]capataz.Opt{},
		func(EventManager) {},
	)

	assert.NoError(t, err)
	t.Run("starts and stops routines in the correct order", func(t *testing.T) {
		AssertExactMatch(t, events,
			[]EventP{
				WorkerStarted("root/child0"),
				WorkerStarted("root/child1"),
				WorkerStarted("root/child2"),
				SupervisorStarted("root"),
				WorkerTerminated("root/child2"),
				WorkerTerminated("root/child1"),
				WorkerTerminated("root/child0"),
				SupervisorTerminated("root"),
			})
	})
}

// Test a supervision tree with three children start and stop in the default
// order (LeftToRight)
func TestStartMutlipleChildrenRightToLeft(t *testing.T) {
	events, err := ObserveSupervisor(
		context.TODO(),
		"root",
		capataz.WithNodes(
			WaitDoneWorker("child0"),
			WaitDoneWorker("child1"),
			WaitDoneWorker("child2"),
		),
		[]capataz.Opt{
			capataz.WithOrder(capataz.RightToLeft),
		},
		func(EventManager) {},
	)

	assert.NoError(t, err)
	t.Run("starts and stops routines in the correct order", func(t *testing.T) {
		AssertExactMatch(t, events,
			[]EventP{
				WorkerStarted("root/child2"),
				WorkerStarted("root/child1"),
				WorkerStarted("root/child0"),
				SupervisorStarted("root"),
				WorkerTerminated("root/child0"),
				WorkerTerminated("root/child1"),
				WorkerTerminated("root/child2"),
				SupervisorTerminated("root"),
			})
	})
}

// Test a supervision tree with two sub-trees start and stop children in the
// default order _always_ (LeftToRight)
func TestStartNestedSupervisors(t *testing.T) {
	parentName := "root"
	b0n := "branch0"
	b1n := "branch1"

	cs := []capataz.Node{
		WaitDoneWorker("child0"),
		WaitDoneWorker("child1"),
		WaitDoneWorker("child2"),
		WaitDoneWorker("child3"),
	}

	b0 := capataz.NewSupervisorSpec(b0n, capataz.WithNodes(cs[0], cs[1]))
	b1 := capataz.NewSupervisorSpec(b1n, capataz.WithNodes(cs[2], cs[3]))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		capataz.WithNodes(
			capataz.Subtree(b0),
			capataz.Subtree(b1),
		),
		[]capataz.Opt{},
		func(EventManager) {},
	)

	assert.NoError(t, err)
	t.Run("starts and stops routines in the correct order", func(t *testing.T) {
		AssertExactMatch(t, events,
			[]EventP{
				// start children from left to right
				WorkerStarted("root/branch0/child0"),
				WorkerStarted("root/branch0/child1"),
				SupervisorStarted("root/branch0"),
				WorkerStarted("root/branch1/child2"),
				WorkerStarted("root/branch1/child3"),
				SupervisorStarted("root/branch1"),
				SupervisorStarted("root"),
				// stops children from right to left
				WorkerTerminated("root/branch1/child3"),
				WorkerTerminated("root/branch1/child2"),
				SupervisorTerminated("root/branch1"),
				WorkerTerminated("root/branch0/child1"),
				WorkerTerminated("root/branch0/child0"),
				SupervisorTerminated("root/branch0"),
				SupervisorTerminated("root"),
			},
		)
	})
}

func TestStartFailedChild(t *testing.T) {
	parentName := "root"
	b0n := "branch0"
	b1n := "branch1"

	cs := []capataz.Node{
		WaitDoneWorker("child0"),
		WaitDoneWorker("child1"),
		WaitDoneWorker("child2"),
		// NOTE: FailStartWorker here
		FailStartWorker("child3"),
		WaitDoneWorker("child4"),
	}

	b0 := capataz.NewSupervisorSpec(b0n, capataz.WithNodes(cs[0], cs[1]))
	b1 := capataz.NewSupervisorSpec(b1n, capataz.WithNodes(cs[2], cs[3], cs[4]))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		capataz.WithNodes(
			capataz.Subtree(b0),
			capataz.Subtree(b1),
		),
		[]capataz.Opt{},
		func(em EventManager) {},
	)

	assert.Error(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/branch0/child0"),
			WorkerStarted("root/branch0/child1"),
			SupervisorStarted("root/branch0"),
			WorkerStarted("root/branch1/child2"),
			//
			// Note child3 fails at this point
			//
			WorkerStartFailed("root/branch1/child3"),
			//
			// After a failure a few things will happen:
			//
			// * The `child4` worker initialization is skipped because of an error on
			// previous sibling
			//
			// * Previous sibling children get stopped in reversed order
			//
			// * The start function returns an error
			//
			WorkerTerminated("root/branch1/child2"),
			SupervisorStartFailed("root/branch1"),
			WorkerTerminated("root/branch0/child1"),
			WorkerTerminated("root/branch0/child0"),
			SupervisorTerminated("root/branch0"),
			SupervisorStartFailed("root"),
		},
	)
}

func TestTerminateFailedChild(t *testing.T) {
	parentName := "root"
	b0n := "branch0"
	b1n := "branch1"

	cs := []capataz.Node{
		WaitDoneWorker("child0"),
		WaitDoneWorker("child1"),
		// NOTE: There is a NeverTerminateWorker here
		NeverTerminateWorker("child2"),
		WaitDoneWorker("child3"),
	}

	b0 := capataz.NewSupervisorSpec(b0n, capataz.WithNodes(cs[0], cs[1]))
	b1 := capataz.NewSupervisorSpec(b1n, capataz.WithNodes(cs[2], cs[3]))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		capataz.WithNodes(
			capataz.Subtree(b0),
			capataz.Subtree(b1),
		),
		[]capataz.Opt{},
		func(em EventManager) {},
	)

	assert.Error(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/branch0/child0"),
			WorkerStarted("root/branch0/child1"),
			SupervisorStarted("root/branch0"),
			WorkerStarted("root/branch1/child2"),
			WorkerStarted("root/branch1/child3"),
			SupervisorStarted("root/branch1"),
			SupervisorStarted("root"),
			// NOTE: From here, the stop of the supervisor begins
			WorkerTerminated("root/branch1/child3"),
			// NOTE: the child2 never stops and fails with a timeout
			WorkerFailed("root/branch1/child2"),
			// NOTE: The supervisor branch1 fails because of child2 timeout
			SupervisorFailed("root/branch1"),
			WorkerTerminated("root/branch0/child1"),
			WorkerTerminated("root/branch0/child0"),
			SupervisorTerminated("root/branch0"),
			SupervisorFailed("root"),
		},
	)
}
