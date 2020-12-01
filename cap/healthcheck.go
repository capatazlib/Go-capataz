package cap

import (
	"time"
)

// HealthReport contains a report for the HealthMonitor
type HealthReport struct {
	failedProcesses         []string
	delayedRestartProcesses []string
}

// HealthyReport represents a healthy report
var HealthyReport = HealthReport{}

// HealthcheckMonitor listens to the events of a supervision tree events, and
// assess if the supervisor is healthy or not
type HealthcheckMonitor struct {
	maxAllowedRestartDuration time.Duration
	maxAllowedFailures        uint32
	failedEvs                 map[string]Event
}

// GetFailedProcesses returns a list of the failed processes
func (hr HealthReport) GetFailedProcesses() []string {
	return hr.failedProcesses
}

// GetDelayedRestartProcesses returns a list of the delayed restart processes
func (hr HealthReport) GetDelayedRestartProcesses() []string {
	return hr.delayedRestartProcesses
}

// IsHealthyReport indicates if this is a healthy report
func (hr HealthReport) IsHealthyReport() bool {
	return len(hr.failedProcesses) == 0 && len(hr.delayedRestartProcesses) == 0
}

// NewHealthcheckMonitor offers a way to monitor a supervision tree health from
// events emitted by it.
// MaxAllowedFailures: the threshold beyond which the environment is considered
//                     unhealthy.
// MaxAllowedRestartDuration: the restart threshold, which if exceeded, indicates
//                            an unhealthy environment. Any process that fails
//                            to restart under the threshold results in an
//                            unhealthy report
func NewHealthcheckMonitor(
	maxAllowedFailures uint32,
	maxAllowedRestartDuration time.Duration,
) HealthcheckMonitor {
	return HealthcheckMonitor{
		maxAllowedRestartDuration: maxAllowedRestartDuration,
		maxAllowedFailures:        maxAllowedFailures,
		failedEvs:                 make(map[string]Event),
	}
}

// HandleEvent is a function that receives supervision events and assess if the
// supervisor sending these events is healthy or not
func (h HealthcheckMonitor) HandleEvent(ev Event) {
	switch ev.GetTag() {
	case ProcessFailed:
		h.failedEvs[ev.GetProcessRuntimeName()] = ev
	case ProcessStarted:
		delete(h.failedEvs, ev.GetProcessRuntimeName())
	}
}

// GetHealthReport returns a string that indicates why a the system
// is unhealthy. Returns empty if everything is ok.
func (h HealthcheckMonitor) GetHealthReport() HealthReport {
	// if there is an acceptable number of failures, things are healthy
	if uint32(len(h.failedEvs)) == 0 {
		return HealthyReport
	}

	hr := HealthReport{
		failedProcesses:         make([]string, 0, len(h.failedEvs)),
		delayedRestartProcesses: make([]string, 0, len(h.failedEvs)),
	}

	// if you have more than maxAllowedFailures process failing, then you are
	// not healthy
	if uint32(len(h.failedEvs)) > h.maxAllowedFailures {
		for processName := range h.failedEvs {
			hr.failedProcesses = append(hr.failedProcesses, processName)
		}
	}

	currentTime := time.Now()
	for processName, ev := range h.failedEvs {
		dur := currentTime.Sub(ev.GetCreated())

		// Capture all failures that are taking too long to recover
		if dur > h.maxAllowedRestartDuration {
			hr.delayedRestartProcesses = append(hr.delayedRestartProcesses, processName)
		}
	}

	return hr
}

// IsHealthy return true when the system is in a healthy state, meaning, no
// processes restarting at the moment
func (h HealthcheckMonitor) IsHealthy() bool {
	return h.GetHealthReport().IsHealthyReport()
}
