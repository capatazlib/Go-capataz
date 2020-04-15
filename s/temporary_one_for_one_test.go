package s_test

//
// NOTE: If you feel it is counter-intuitive to have workers start before
// supervisors in the assertions bellow, check stest/README.md
//

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/capatazlib/go-capataz/internal/stest"

	"github.com/capatazlib/go-capataz/s"
)

func TestTemporaryOneForOneSingleFailingWorkerDoesNotRecover(t *testing.T) {
	parentName := "root"
	// Fail only one time
	worker1, failWorker1 := FailOnSignalWorker(1, "worker1", s.WithRestart(s.Temporary))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		s.WithChildren(worker1),
		[]s.Opt{},
		func(em EventManager) {
			// NOTE: we won't stop the supervisor until the child has failed at least
			// once
			evIt := em.Iterator()
			// 1) Wait till all the tree is up
			evIt.SkipTill(SupervisorStarted("root"))
			// 2) Start the failing behavior of worker1
			failWorker1(true /* done */)
			// 3) Wait till first restart
			evIt.SkipTill(WorkerFailed("root/worker1"))
		},
	)

	assert.NoError(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/worker1"),
			SupervisorStarted("root"),
			// ^^^ 1) failWorker1 starts executing here
			WorkerFailed("root/worker1"),
			// ^^^ 2) We see the failure, and then nothing else of this child
			SupervisorTerminated("root"),
		},
	)
}

func TestTemporaryOneForOneNestedFailingWorkerDoesNotRecover(t *testing.T) {
	parentName := "root"
	// Fail only one time
	worker1, failWorker1 := FailOnSignalWorker(1, "worker1", s.WithRestart(s.Temporary))
	tree1 := s.New("subtree1", s.WithChildren(worker1))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		s.WithChildren(s.Subtree(tree1)),
		[]s.Opt{},
		func(em EventManager) {
			// NOTE: we won't stop the supervisor until the child has failed at least
			// once
			evIt := em.Iterator()
			// 1) Wait till all the tree is up
			evIt.SkipTill(SupervisorStarted("root"))
			// 2) Start the failing behavior of worker1
			failWorker1(true /* done */)
			// 3) Wait till first restart
			evIt.SkipTill(WorkerFailed("root/subtree1/worker1"))
		},
	)

	assert.NoError(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/subtree1/worker1"),
			SupervisorStarted("root/subtree1"),
			SupervisorStarted("root"),
			// ^^^ 1) Wait till root starts
			WorkerFailed("root/subtree1/worker1"),
			// ^^^ 2) We see the failure, and then nothing else of this child
			SupervisorTerminated("root/subtree1"),
			SupervisorTerminated("root"),
		},
	)
}

func TestTemporaryOneForOneSingleCompleteWorkerDoesNotRestart(t *testing.T) {
	parentName := "root"
	// Fail only one time
	worker1, completeWorker1 := CompleteOnSignalWorker(1, "worker1", s.WithRestart(s.Temporary))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		s.WithChildren(worker1),
		[]s.Opt{},
		func(em EventManager) {
			// NOTE: we won't stop the supervisor until the child has failed at least
			// once
			evIt := em.Iterator()
			// 1) Wait till all the tree is up
			evIt.SkipTill(SupervisorStarted("root"))
			// 2) Start the complete behavior of worker1
			completeWorker1()
			// 3) Wait till first restart
			evIt.SkipTill(WorkerCompleted("root/worker1"))
		},
	)

	assert.NoError(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/worker1"),
			SupervisorStarted("root"),
			// ^^^ 1) completeWorker1 starts executing here
			WorkerCompleted("root/worker1"),
			// ^^^ 2) We see completion, and then nothing else of this child
			SupervisorTerminated("root"),
		},
	)
}

func TestTemporaryOneForOneNestedCompleteWorkerDoesNotRestart(t *testing.T) {
	parentName := "root"
	// Fail only one time
	worker1, completeWorker1 := CompleteOnSignalWorker(1, "worker1", s.WithRestart(s.Temporary))
	tree1 := s.New("subtree1", s.WithChildren(worker1))

	events, err := ObserveSupervisor(
		context.TODO(),
		parentName,
		s.WithChildren(s.Subtree(tree1)),
		[]s.Opt{},
		func(em EventManager) {
			// NOTE: we won't stop the supervisor until the child has failed at least
			// once
			evIt := em.Iterator()
			// 1) Wait till all the tree is up
			evIt.SkipTill(SupervisorStarted("root"))
			// 2) Start the failing behavior of worker1
			completeWorker1()
			// 3) Wait till first restart
			evIt.SkipTill(WorkerCompleted("root/subtree1/worker1"))
		},
	)

	assert.NoError(t, err)

	AssertExactMatch(t, events,
		[]EventP{
			// start children from left to right
			WorkerStarted("root/subtree1/worker1"),
			SupervisorStarted("root/subtree1"),
			SupervisorStarted("root"),
			// ^^^ 1) completeWorker1 starts executing here
			WorkerCompleted("root/subtree1/worker1"),
			// ^^^ 2) We see completion, and then nothing else of this child
			SupervisorTerminated("root/subtree1"),
			SupervisorTerminated("root"),
		},
	)
}
