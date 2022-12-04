// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"fmt"
	"time"

	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/go-nagios"
	"github.com/rs/zerolog"
)

func appendPerfData(exitState *nagios.ExitState, start time.Time, logger zerolog.Logger) {
	// Record plugin runtime, emit this metric regardless of exit
	// point/cause.
	runtimeMetric := nagios.PerformanceData{
		Label: "time",
		Value: fmt.Sprintf("%dms", time.Since(start).Milliseconds()),
	}
	if err := exitState.AddPerfData(false, runtimeMetric); err != nil {
		logger.Error().
			Err(err).
			Msg("failed to add time (runtime) performance data metric")
	}

}

// getPerfData gathers performance data metrics that we wish to report.
func getPerfData(
	allAssertions restart.RebootRequiredAsserters,
	fileAssertions restart.RebootRequiredAsserters,
	registryAssertions restart.RebootRequiredAsserters,
) []nagios.PerformanceData {

	return []nagios.PerformanceData{
		// The `time` (runtime) metric is appended at plugin exit, so do not
		// duplicate it here.
		{
			Label: "evaluated_assertions",
			Value: fmt.Sprintf("%d", len(allAssertions)),
		},
		{
			Label: "evaluated_file_assertions",
			Value: fmt.Sprintf("%d", len(fileAssertions)),
		},
		{
			Label: "evaluated_registry_assertions",
			Value: fmt.Sprintf("%d", len(registryAssertions)),
		},
		{
			Label: "matched_assertions",
			Value: fmt.Sprintf("%d", allAssertions.NumMatched()),
		},
		{
			Label: "ignored_assertions",
			Value: fmt.Sprintf("%d", allAssertions.NumIgnored()),
		},
		{
			Label: "errors",
			Value: fmt.Sprintf("%d", allAssertions.NumErrors(false)),
		},
	}

}
