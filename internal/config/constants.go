// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package config

const myAppName string = "check-restart"
const myAppURL string = "https://github.com/atc0005/check-restart"

// ExitCodeCatchall indicates a general or miscellaneous error has occurred.
// This exit code is not directly used by monitoring plugins in this project.
// See https://tldp.org/LDP/abs/html/exitcodes.html for additional details.
const ExitCodeCatchall int = 1

const (
	versionFlagHelp               string = "Whether to display application version and then immediately exit application."
	logLevelFlagHelp              string = "Sets log level."
	brandingFlagHelp              string = "Toggles emission of branding details with plugin status details. This output is disabled by default."
	verboseOutputFlagHelp         string = "Toggles emission of detailed output. This level of output is disabled by default."
	showIgnoredFlagHelp           string = "Toggles emission of ignored assertion matches in the final plugin output. This is disabled by default."
	disableDefaultIgnoredFlagHelp string = "Disables use of default ignored assertion path entries."
)

// shorthandFlagSuffix is appended to short flag help text to emphasize that
// the flag is a shorthand version of a longer flag.
const shorthandFlagSuffix = " (shorthand)"

// Flag names for consistent references. Exported so that they're available
// from tests.
const (
	// HelpFlagLong      string = "help"
	// HelpFlagShort     string = "h"
	// VersionFlagShort  string = "v"

	VersionFlagLong                string = "version"
	VerboseFlagLong                string = "verbose"
	VerboseFlagShort               string = "v"
	BrandingFlag                   string = "branding"
	TimeoutFlagLong                string = "timeout"
	TimeoutFlagShort               string = "t"
	ShowIgnoredFlagLong            string = "show-ignored"
	ShowIgnoredFlagShort           string = "si"
	DisableDefaultIgnoredFlagShort string = "dd"
	DisableDefaultIgnoredFlagLong  string = "disable-default-ignored"
	LogLevelFlagLong               string = "log-level"
	LogLevelFlagShort              string = "ll"
)

// Default flag settings if not overridden by user input
const (
	defaultLogLevel              string = "info"
	defaultBranding              bool   = false
	defaultVerboseOutput         bool   = false
	defaultShowIgnored           bool   = false
	defaultDisableDefaultIgnored bool   = false
	defaultDisplayVersionAndExit bool   = false
)

const (
	appTypePlugin    string = "plugin"
	appTypeInspector string = "inspector"
)
