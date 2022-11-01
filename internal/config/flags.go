// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package config

import (
	"flag"
	"fmt"
	"os"
)

// supportedValuesFlagHelpText is a flag package helper function that combines
// base help text with a list of supported values for the flag.
func supportedValuesFlagHelpText(baseHelpText string, supportedValues []string) string {
	return fmt.Sprintf(
		"%s Supported values: %v",
		baseHelpText,
		supportedValues,
	)
}

// handleFlagsConfig handles toggling the exposure of specific configuration
// flags to the user. This behavior is controlled via the specified
// application type as set by each cmd. Based on the application type, a
// smaller subset of flags specific to each type are exposed along with a set
// common to all application types.
func (c *Config) handleFlagsConfig(appType AppType) {

	var (
		// Application specific template used for generating lead-in
		// usage/help text.
		usageTextHeaderTmpl string

		// Additional requirements for using positional arguments. May not
		// apply to all application types.
		positionalArgRequirements string

		// A human readable description of the specific application.
		appDescription string
	)

	// Flags specific to one application type or the other
	switch {
	case appType.Plugin:

		// Override the default Help output with a brief lead-in summary of
		// the expected syntax and project version.
		//
		// For this specific application type, flags are *required*.
		//
		// https://stackoverflow.com/a/36787811/903870
		// https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html
		usageTextHeaderTmpl = "%s\n\nUsage:  %s <flags>\n\n%s\n\nFlags:\n"

		appDescription = "Nagios plugin used to monitor for the need to reboot a system or services."

		flag.BoolVar(&c.EmitBranding, BrandingFlag, defaultBranding, brandingFlagHelp)

		flag.BoolVar(&c.VerboseOutput, VerboseFlagShort, defaultVerboseOutput, verboseOutputFlagHelp+" (shorthand)")
		flag.BoolVar(&c.VerboseOutput, VerboseFlagLong, defaultVerboseOutput, verboseOutputFlagHelp)

		flag.BoolVar(&c.ShowIgnored, ShowIgnoredFlagShort, defaultShowIgnored, showIgnoredFlagHelp+" (shorthand)")
		flag.BoolVar(&c.ShowIgnored, ShowIgnoredFlagLong, defaultShowIgnored, showIgnoredFlagHelp)

	case appType.Inspecter:

		// Override the default Help output with a brief lead-in summary of
		// the expected syntax and project version.
		//
		// For this specific application type, flags are required unless the
		// host/url pattern is provided, at which point flags are optional.
		// Because I'm not sure how to specify this briefly, both are listed
		// as optional.
		//
		// https://stackoverflow.com/a/36787811/903870
		// https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html
		usageTextHeaderTmpl = "%s\n\nUsage:  %s [flags] [pattern]\n\n%s\n\nFlags:\n"

		// positionalArgRequirements = fmt.Sprintf(
		// 	"\nPositional Argument (\"pattern\") Requirements:\n\n"+
		// 		"- if the %q or %q"+
		// 		" flags are specified, the URL pattern is ignored"+
		// 		"\n- if the %q flag is specified, its value will be"+
		// 		" ignored if a port is provided in the given URL pattern",
		// 	ServerFlagLong,
		// 	FilenameFlagLong,
		// 	PortFlagLong,
		// )

		appDescription = "Used to generate a summary of metadata indicating the need to reboot a system or services."

		flag.BoolVar(&c.VerboseOutput, VerboseFlagShort, defaultVerboseOutput, verboseOutputFlagHelp+" (shorthand)")
		flag.BoolVar(&c.VerboseOutput, VerboseFlagLong, defaultVerboseOutput, verboseOutputFlagHelp)

	}

	// Shared flags for all application type

	flag.StringVar(
		&c.LoggingLevel,
		LogLevelFlagShort,
		defaultLogLevel,
		supportedValuesFlagHelpText(logLevelFlagHelp, supportedLogLevels())+" (shorthand)",
	)
	flag.StringVar(
		&c.LoggingLevel,
		LogLevelFlagLong,
		defaultLogLevel,
		supportedValuesFlagHelpText(logLevelFlagHelp, supportedLogLevels()),
	)

	flag.BoolVar(&c.ShowVersion, VersionFlagLong, defaultDisplayVersionAndExit, versionFlagHelp)

	// Prepend a brief lead-in summary of the expected syntax and project
	// version before emitting the default Help output.
	//
	// https://stackoverflow.com/a/36787811/903870
	// https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html
	flag.Usage = func() {
		headerText := fmt.Sprintf(
			usageTextHeaderTmpl,
			Version(),
			os.Args[0],
			appDescription,
		)

		footerText := fmt.Sprintf(
			"\nSee project README at %s for examples and additional details.\n",
			myAppURL,
		)

		// Override default of stderr as destination for help output. This
		// allows Nagios XI and similar monitoring systems to call plugins
		// with the `--help` flag and have it display within the Admin web UI.
		flag.CommandLine.SetOutput(os.Stdout)

		fmt.Fprintln(flag.CommandLine.Output(), headerText)
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), positionalArgRequirements)
		fmt.Fprintln(flag.CommandLine.Output(), footerText)
	}

	// parse flag definitions from the argument list
	flag.Parse()

}
