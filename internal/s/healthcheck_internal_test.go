package s

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHealthNothingToDo(t *testing.T) {

	healthcheckMonitor := NewHealthcheckMonitor(0, 0*time.Millisecond)

	assert.True(t, healthcheckMonitor.IsHealthy())
}

func TestHealthHappyPath(t *testing.T) {
	healthcheckMonitor := NewHealthcheckMonitor(0, 0*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())
	notifier.workerStarted("w2", time.Now())
	assert.True(t, healthcheckMonitor.IsHealthy())
}

func TestAtMaxFailuresAndUnderRestartDuration(t *testing.T) {
	healthcheckMonitor := NewHealthcheckMonitor(2, 1000*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())
	notifier.workerStarted("w2", time.Now())
	assert.True(t, healthcheckMonitor.IsHealthy())

	// We tolerate 2 failures, so OK
	notifier.workerFailed("w1", errors.New("w1 error"))
	assert.True(t, healthcheckMonitor.IsHealthy())

	// We tolerate 2 failures and this is #2, so OK
	notifier.workerFailed("w2", errors.New("w2 error"))
	assert.True(t, healthcheckMonitor.IsHealthy())
}

func TestHealthyReport(t *testing.T) {
	healthcheckMonitor := NewHealthcheckMonitor(0, 0*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())

	hr := healthcheckMonitor.GetHealthReport()
	assert.True(t, hr.IsHealthyReport())
}

func TestUnhealthyFailuresReport(t *testing.T) {
	// Don't tolerate any failures
	healthcheckMonitor := NewHealthcheckMonitor(0, 1000*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())
	// Unacceptable failure
	notifier.workerFailed("w1", errors.New("w1 error"))

	hr := healthcheckMonitor.GetHealthReport()
	assert.False(t, hr.IsHealthyReport())

	// Failures are over tolerance
	assert.EqualValues(t, 1, len(hr.GetFailedProcesses()))
	assert.True(t, hr.GetFailedProcesses()["w1"])
	// restart delays are under tolerance
	assert.EqualValues(t, 0, len(hr.GetDelayedRestartProcesses()))
}

func TestUnhealthyDelaysReport(t *testing.T) {
	// Do not tolerate any restart delay
	healthcheckMonitor := NewHealthcheckMonitor(100, 0*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())
	// Unacceptable delay
	notifier.workerFailed("w1", errors.New("w1 error"))

	hr := healthcheckMonitor.GetHealthReport()
	assert.False(t, hr.IsHealthyReport())

	// Failures are under tolerance
	assert.EqualValues(t, 0, len(hr.GetFailedProcesses()))
	// restart delays are over tolerance
	assert.EqualValues(t, 1, len(hr.GetDelayedRestartProcesses()))
	assert.True(t, hr.GetDelayedRestartProcesses()["w1"])
}

func TestHealthRestoredReport(t *testing.T) {
	// Do not tolerate any failures or restart delays
	healthcheckMonitor := NewHealthcheckMonitor(0, 0*time.Millisecond)

	var notifier EventNotifier = func(ev Event) {
		healthcheckMonitor.HandleEvent(ev)
	}

	notifier.workerStarted("w1", time.Now())
	// Unacceptable failures and delays
	notifier.workerFailed("w1", errors.New("w1 error"))

	hr := healthcheckMonitor.GetHealthReport()
	// Failures are over tolerance
	assert.False(t, hr.IsHealthyReport())

	// Failures are over tolerance
	assert.EqualValues(t, 1, len(hr.GetFailedProcesses()))
	assert.True(t, hr.GetFailedProcesses()["w1"])
	// restart delays are over tolerance
	assert.EqualValues(t, 1, len(hr.GetDelayedRestartProcesses()))
	assert.True(t, hr.GetDelayedRestartProcesses()["w1"])

	// Failures recovered
	notifier.workerStarted("w1", time.Now())
	assert.True(t, healthcheckMonitor.GetHealthReport().IsHealthyReport())
}
