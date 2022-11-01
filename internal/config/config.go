// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Updated via Makefile builds. Setting placeholder value here so that
// something resembling a version string will be provided for non-Makefile
// builds.
var version = "x.y.z"

// ErrVersionRequested indicates that the user requested application version
// information.
var ErrVersionRequested = errors.New("version information requested")

// ErrUnsupportedOption indicates that an unsupported option was specified.
var ErrUnsupportedOption = errors.New("unsupported option")

// AppType represents the type of application that is being
// configured/initialized. Not all application types will use the same
// features and as a result will not accept the same flags. Unless noted
// otherwise, each of the application types are incompatible with each other,
// though some flags are common to all types.
type AppType struct {

	// Plugin represents an application used as a Nagios plugin.
	Plugin bool

	// Inspecter represents an application used for one-off or isolated
	// checks. Unlike a Nagios plugin which is focused on specific attributes
	// resulting in a severity-based outcome, an Inspecter application is
	// intended for examining a small set of targets for
	// informational/troubleshooting purposes.
	Inspecter bool
}

// Config represents the application configuration as specified via
// command-line flags.
type Config struct {

	// LoggingLevel is the supported logging level for this application.
	LoggingLevel string

	// EmitBranding controls whether "generated by" text is included at the
	// bottom of application output. This output is included in the Nagios
	// dashboard and notifications. This output may not mix well with branding
	// output from other tools such as atc0005/send2teams which also insert
	// their own branding output.
	EmitBranding bool

	// VerboseOutput controls whether detailed output is emitted along with
	// standard information.
	VerboseOutput bool

	// ShowVersion is a flag indicating whether the user opted to display only
	// the version string and then immediately exit the application.
	ShowVersion bool

	// ShowIgnored is a flag indicating whether the user opted to include
	// assertion matches that are marked as ignored in the final plugin
	// output.
	ShowIgnored bool

	// DisableDefaultIgnored is a flag indicating whether the user opted to
	// disable the default ignored paths used during filtering to mark
	// matching assertion path entries as ignored in the final plugin output.
	DisableDefaultIgnored bool

	// Log is an embedded zerolog Logger initialized via config.New().
	Log zerolog.Logger
}

// Version emits application name, version and repo location.
func Version() string {
	return fmt.Sprintf("%s %s (%s)", myAppName, version, myAppURL)
}

// Branding accepts a message and returns a function that concatenates that
// message with version information. This function is intended to be called as
// a final step before application exit after any other output has already
// been emitted.
func Branding(msg string) func() string {
	return func() string {
		return strings.Join([]string{msg, Version()}, "")
	}
}

// New is a factory function that produces a new Config object based on user
// provided flag and config file values. It is responsible for validating
// user-provided values and initializing the logging settings used by this
// application.
func New(appType AppType) (*Config, error) {
	var config Config

	config.handleFlagsConfig(appType)

	if config.ShowVersion {
		return nil, ErrVersionRequested
	}

	if err := config.validate(appType); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// initialize logging just as soon as validation is complete
	if err := config.setupLogging(appType); err != nil {
		return nil, fmt.Errorf(
			"failed to set logging configuration: %w",
			err,
		)
	}

	return &config, nil

}
