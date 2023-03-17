// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package reports

import (
	"fmt"
	"io"
	"strings"

	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/go-nagios"
)

// CheckRebootOneLineSummary returns a one-line summary of the evaluation
// results suitable for display and notification purposes. A boolean value is
// accepted which indicates whether assertion values marked as ignored (during
// filtering) should also be considered.
func CheckRebootOneLineSummary(assertions restart.RebootRequiredAsserters, evalIgnored bool) string {
	var summary string

	switch {

	// We're not checking whether errors were encountered at this point, just
	// whether a successful determination has been made that a reboot is
	// needed.
	case assertions.RebootRequired():
		summary = fmt.Sprintf(
			"%s: Reboot needed (assertions: %d applied, %d matched, %d ignored)",
			assertions.ServiceState().Label,
			assertions.NumApplied(),
			assertions.NumMatched(),
			assertions.NumIgnored(),
		)

	// Errors have occurred which prevent accurately detecting whether a
	// reboot is needed.
	case assertions.HasErrors(evalIgnored):
		summary = fmt.Sprintf(
			// "%s: Reboot evaluation failed due to errors (applied %d reboot assertions, %d errors occurred)",
			"%s: Reboot evaluation failed; %d errors (assertions: %d applied, %d matched, %d ignored)",
			assertions.ServiceState().Label,
			assertions.NumErrors(evalIgnored),
			assertions.NumApplied(),
			assertions.NumMatched(),
			assertions.NumIgnored(),
		)

	case assertions.IsOKState():
		summary = fmt.Sprintf(
			"%s: Reboot not needed (assertions: %d applied, %d matched, %d ignored)",
			assertions.ServiceState().Label,
			assertions.NumApplied(),
			assertions.NumMatched(),
			assertions.NumIgnored(),
		)

	default:
		summary = "BUG: Expected assertions collection state unexpected"

	}

	return summary

}

//nolint:all
//lint:ignore U1000 disabling use per GH-119, but may re-enable later via flag
func writeReportHeader(w io.Writer, assertions restart.RebootRequiredAsserters, verbose bool) {
	fmt.Fprintf(
		w,
		"%[1]sSummary:%[1]s%[1]s",
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		w,
		"  - %d total reboot assertions applied%s",
		assertions.NumApplied(),
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		w,
		"  - %d total reboot assertions matched%s",
		assertions.NumMatched(),
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		w,
		"  - %d total reboot assertions ignored%s",
		assertions.NumIgnored(),
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		w,
		"%[1]s%[2]s%[1]s%[1]s",
		nagios.CheckOutputEOL,
		strings.Repeat("-", 50),
	)
}

func writeAssertions(w io.Writer, assertions restart.RebootRequiredAsserters, verbose bool) {

	// Specific "template" strings used to control formatting/indentation
	// levels for the first item in a listing and any "sub details" associated
	// with the item. The formatting is intended to convey this relationship
	// at a glance.
	const topDetailTemplateStr = "\n  - %s%s"
	const subDetailTemplateStr = "    %s%s"

	// We don't filter on whether the assertion is ignored as we're using this
	// helper function to process collections of both types.
	for _, assertion := range assertions {
		if !assertion.HasEvidence() {
			continue
		}

		// While there is *usually* one reason for a reboot, the current
		// design allows for multiple reasons.
		for _, reason := range assertion.RebootReasons() {
			fmt.Fprintf(w, topDetailTemplateStr, reason, nagios.CheckOutputEOL)

			// We are processing types beneath RebootReasons so that we can
			// emit more detailed information following a "standard" reboot
			// required statement.
			if verbose {
				appendAdditionalContext(w, assertion, subDetailTemplateStr)
			}

		}
	}

	fmt.Fprint(w, nagios.CheckOutputEOL)
}

// CheckRebootReport returns a formatted report of the evaluation results
// suitable for display and notification purposes. If specified, additional
// details are provided.
func CheckRebootReport(assertions restart.RebootRequiredAsserters, showIgnored bool, verbose bool) string {
	var report strings.Builder

	// Disabling per GH-119, but may re-enable later via flag.
	//
	// writeReportHeader(&report, assertions, verbose)

	switch {

	case assertions.RebootRequired():

		// writeMatchedPaths(&report, assertions, verbose)

		fmt.Fprintf(
			&report,
			"Reboot required because:%[1]s",
			nagios.CheckOutputEOL,
		)

		notIgnoredAssertions := assertions.NotIgnoredItems()

		logger.Printf("%d notIgnoredAssertions to process", len(notIgnoredAssertions))

		writeAssertions(&report, notIgnoredAssertions, verbose)

	case assertions.IsOKState():
		fmt.Fprintf(&report, "Reboot not required%s", nagios.CheckOutputEOL)

	}

	if assertions.HasIgnored() && showIgnored {
		fmt.Fprintf(
			&report,
			"%[1]sAssertions ignored:%[1]s",
			nagios.CheckOutputEOL,
		)

		ignoredAssertions := assertions.IgnoredItems()

		logger.Printf("%d ignoredAssertions to process", len(ignoredAssertions))

		writeAssertions(&report, ignoredAssertions, verbose)
	}

	// Normalize output so that Windows-specific paths are less likely to be
	// mangled when included in generated notifications.
	return substituteSeparators(report.String())

}

func appendAdditionalContext(
	w io.Writer,
	assertion restart.RebootRequiredAsserter,
	subDetailTemplateStr string,
) {

	switch v := assertion.(type) {
	case restart.RebootRequiredAsserterWithSubPaths:

		if v.HasSubPathMatches() {
			logger.Printf("%q has subpath evidence", assertion.String())

			for _, path := range v.MatchedPaths() {
				fmt.Fprintf(
					w,
					subDetailTemplateStr,
					"subpath: "+path.Base(),
					nagios.CheckOutputEOL,
				)
			}

		}

	default:

		logger.Printf("%q does not have subkey evidence", assertion.String())
	}

	switch v := assertion.(type) {
	case restart.RebootRequiredAsserterWithDataDisplay:
		logger.Printf("Type assertion worked, value available for check result")

		fmt.Fprintf(
			w,
			subDetailTemplateStr,
			v.DataDisplay(),
			nagios.CheckOutputEOL,
		)

	default:
		logger.Printf("Type assertion failed, value not available for check result")
		logger.Printf("Type found: %T", v)
	}

}

// substitutePathSeparators replaces backslashes used as Windows-specific
// registry and file path separators (even the escaped backslash) with a
// single slash.
func substituteSeparators(input string) string {
	input = strings.ReplaceAll(input, `\\`, `\`)
	input = strings.ReplaceAll(input, `\`, `/`)

	return input
}
