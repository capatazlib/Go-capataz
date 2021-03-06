package main

import (
	"os"

	"github.com/capatazlib/go-capataz/cap"
	"github.com/sirupsen/logrus"
)

func newLogEventNotifier() (*logrus.Entry, cap.EventNotifier) {
	log := logrus.New()
	log.Out = os.Stdout
	log.Level = logrus.DebugLevel
	log.SetFormatter(&logrus.JSONFormatter{})

	ll := log.WithFields(logrus.Fields{})

	return ll, func(ev cap.Event) {
		if ev.Err() != nil {
			ll = log.WithError(ev.Err())
		}
		ll.WithFields(logrus.Fields{
			"process_runtime_name": ev.GetProcessRuntimeName(),
			"created_at":           ev.GetCreated(),
		}).Debug(ev.GetTag().String())
	}
}
