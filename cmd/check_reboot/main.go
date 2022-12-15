// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/atc0005/check-restart/internal/config"
	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/check-restart/internal/restart/files"
	"github.com/atc0005/check-restart/internal/restart/registry"
	"github.com/atc0005/check-restart/internal/restart/reports"
	"github.com/atc0005/go-nagios"

	"github.com/rs/zerolog"
)

func main() {

	plugin := nagios.NewPlugin()

	// defer this from the start so it is the last deferred function to run
	defer plugin.ReturnCheckResults()

	// Setup configuration by parsing user-provided flags.
	cfg, cfgErr := config.New(config.AppType{Plugin: true})
	switch {
	case errors.Is(cfgErr, config.ErrVersionRequested):
		fmt.Println(config.Version())

		return

	case cfgErr != nil:

		// We make some assumptions when setting up our logger as we do not
		// have a working configuration based on sysadmin-specified choices.
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true}
		logger := zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()

		logger.Err(cfgErr).Msg("Error initializing application")

		plugin.ServiceOutput = fmt.Sprintf(
			"%s: Error initializing application",
			nagios.StateCRITICALLabel,
		)
		plugin.AddError(cfgErr)
		plugin.ExitStatusCode = nagios.StateCRITICALExitCode

		return
	}

	if cfg.EmitBranding {
		// If enabled, show application details at end of notification
		plugin.BrandingCallback = config.Branding("Notification generated by ")
	}

	handleLibraryLogging()

	log := cfg.Log.With().Logger()

	log.Debug().Msg("Retrieving default registry reboot assertions")
	registryAssertions := registry.DefaultRebootRequiredAssertions()
	log.Debug().
		Int("registry_assertions", len(registryAssertions)).
		Msg("Retrieved default registry reboot assertions")

	log.Debug().Msg("Retrieving default file reboot assertions")
	fileAssertions := files.DefaultRebootRequiredAssertions()
	log.Debug().
		Int("file_assertions", len(fileAssertions)).
		Msg("Retrieved default file reboot assertions")

	log.Debug().Msg("Finished retrieving reboot assertions")

	allAssertions := make(restart.RebootRequiredAsserters, 0, len(registryAssertions)+len(fileAssertions))
	allAssertions = append(allAssertions, registryAssertions...)
	allAssertions = append(allAssertions, fileAssertions...)

	log.Debug().
		Int("all_assertions", len(allAssertions)).
		Msg("All assertions retrieved")

	log.Debug().Msg("Validating assertions collection")
	if err := allAssertions.Validate(); err != nil {
		log.Error().Err(err).Msg("Failed to validate provided assertions")

		plugin.AddError(err)
		plugin.ExitStatusCode = nagios.StateCRITICALExitCode
		plugin.ServiceOutput = fmt.Sprintf(
			"%s: Failed to validate list of reboot evaluations",
			nagios.StateCRITICALLabel,
		)

		return
	}

	log.Debug().Msg("Evaluating reboot assertions")
	allAssertions.Evaluate()

	applyIgnorePatterns(allAssertions, cfg.DisableDefaultIgnored, log)

	pd := getPerfData(allAssertions, fileAssertions, registryAssertions)
	if err := plugin.AddPerfData(false, pd...); err != nil {
		log.Error().
			Err(err).
			Msg("failed to add performance data")
	}

	switch {
	case !allAssertions.IsOKState():

		log.Debug().Msg("case !allAssertions.IsOKState() triggered")

		if allAssertions.RebootRequired() {

			// If emitted by default NSClient++ will send back stderr and
			// stdout blended together.
			//
			// The standard deployment procedure (if emitting this at Error
			// level) will likely become explicitly disabling logging entirely
			// in order to avoid this message displaying within the Nagios web
			// UI and notifications by default.
			//
			// Because it would be beneficial to have logging enabled by
			// default and left on by the sysadmin, we need to ensure that only
			// "real" issues are emitted by default.
			log.Debug().
				Int("assertions_applied", allAssertions.NumApplied()).
				Int("assertions_matched", allAssertions.NumMatched()).
				Int("assertions_ignored", allAssertions.NumIgnored()).
				Msg("Reboot assertions matched, reboot needed")

			plugin.AddError(restart.ErrRebootRequired)
		}

		log.Debug().Msg("allAssertions.RebootRequired() NOT triggered")

		// Include all errors collected during evaluation. Don't include
		// errors from assertions marked as ignored.
		if allAssertions.HasErrors(false) {
			log.Error().
				Int("assertions_applied", allAssertions.NumApplied()).
				Int("assertions_matched", allAssertions.NumMatched()).
				Int("assertions_ignored", allAssertions.NumIgnored()).
				Int("errors", allAssertions.NumErrors(false)).
				Msg("Errors encountered evaluating need for reboot")

			plugin.AddError(allAssertions.Errs(false)...)
		}

		log.Debug().Msg("allAssertions.HasErrors(false) NOT triggered")

		plugin.ExitStatusCode = allAssertions.ServiceState().ExitCode
		plugin.ServiceOutput = reports.CheckRebootOneLineSummary(allAssertions, false)
		plugin.LongServiceOutput = reports.CheckRebootReport(allAssertions, cfg.ShowIgnored, cfg.VerboseOutput)

		return

	default:

		log.Debug().Msg("default case for overall plugin state triggered")

		log.Debug().
			Int("num_reboot_assertions_applied", allAssertions.NumApplied()).
			Int("num_reboot_assertions_matched", allAssertions.NumMatched()).
			Msg("No (non-ignored) reboot assertions matched")

		plugin.ServiceOutput = reports.CheckRebootOneLineSummary(allAssertions, false)
		plugin.LongServiceOutput = reports.CheckRebootReport(allAssertions, cfg.ShowIgnored, cfg.VerboseOutput)
		plugin.ExitStatusCode = allAssertions.ServiceState().ExitCode

		return

	}

}
