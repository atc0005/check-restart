// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/check-restart/internal/restart/files"
	"github.com/atc0005/check-restart/internal/restart/registry"
	"github.com/rs/zerolog"
)

func applyIgnorePatterns(
	allAssertions restart.RebootRequiredAsserters,
	disableDefaultIgnored bool,
	logger zerolog.Logger,
) {
	switch {
	case disableDefaultIgnored:
		logger.Debug().Msg("Skipping use of default ignored path entries for reboot assertions")
	default:
		logger.Debug().Msg("Retrieving default ignored path entries for registry reboot assertions")
		registryignorePatterns := registry.DefaultRebootRequiredIgnoredPaths()
		logger.Debug().
			Int("registry_ignore_patterns", len(registryignorePatterns)).
			Msg("Retrieved default registry ignore path patterns")

		logger.Debug().Msg("Retrieving default ignored path entries for file assertions")
		fileignorePatterns := files.DefaultRebootRequiredIgnoredPaths()
		logger.Debug().
			Int("file_ignore_patterns", len(fileignorePatterns)).
			Msg("Retrieved default file ignore path patterns")

		logger.Debug().Msg("Finished retrieving default ignored path entries")

		allIgnorePatterns := make([]string, 0, len(registryignorePatterns)+len(fileignorePatterns))
		allIgnorePatterns = append(allIgnorePatterns, registryignorePatterns...)
		allIgnorePatterns = append(allIgnorePatterns, fileignorePatterns...)

		logger.Debug().Msg("Filtering reboot assertions")

		allAssertions.Filter(allIgnorePatterns)
	}
}
