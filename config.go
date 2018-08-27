package phrasegen

import (
	"github.com/airbrake/gobrake"
	"github.com/sirupsen/logrus"
)

var (
	// Version of application
	Version string

	// Revision of application
	Revision string

	// Environment of application
	Environment string

	// Notifier is an app-wide error notifier
	Notifier *gobrake.Notifier

	// Logger is an app-wide logger
	Logger *logrus.Logger

	// Config is a settings for the application
	Config Configuration
)

// Configuration of the file
type Configuration struct {
	ErrbitHost string
	ErrbitID   int64
	ErrbitKey  string
}

/*
	"github.com/namsral/flag"
*/
