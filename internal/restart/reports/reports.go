// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package reports

import (
	"fmt"
	"strings"

	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/go-nagios"
)

// CheckRebootOneLineSummary returns a one-line summary of the evaluation
// results suitable for display and notification purposes.
func CheckRebootOneLineSummary(rcr restart.RebootCheckResults) string {
	var summary string

	switch {

	// We're not checking whether errors were encountered at this point, just
	// whether a successful determination has been made that a reboot is
	// needed.
	case rcr.RebootRequired():
		summary = fmt.Sprintf(
			"%s: Reboot needed (applied %d reboot assertions, %d matched)",
			rcr.ServiceState().Label,
			rcr.RebootAssertionsApplied(),
			rcr.RebootAssertionsMatched(),
		)

	// Errors have occurred which prevent accurately detecting whether a
	// reboot is needed.
	case rcr.HasErrors():
		summary = fmt.Sprintf(
			"%s: Reboot evaluation failed due to errors (applied %d reboot assertions, %d errors occurred)",
			rcr.ServiceState().Label,
			rcr.RebootAssertionsApplied(),
			rcr.NumErrors(),
		)

	case rcr.IsOKState():
		summary = fmt.Sprintf(
			"%s: Reboot not needed (applied %d reboot assertions, %d matched)",
			rcr.ServiceState().Label,
			rcr.RebootAssertionsApplied(),
			rcr.RebootAssertionsMatched(),
		)

	}

	return summary

}

// CheckRebootReport returns a formatted report of the evaluation results
// suitable for display and notification purposes. If specified, additional
// details are provided.
func CheckRebootReport(rcr restart.RebootCheckResults, verbose bool) string {
	var report strings.Builder

	fmt.Fprintf(
		&report,
		"%[1]sSummary:%[1]s%[1]s",
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		&report,
		"  - %d total reboot assertions applied%s",
		rcr.RebootAssertionsApplied(),
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		&report,
		"  - %d total reboot assertions matched%s",
		rcr.RebootAssertionsMatched(),
		nagios.CheckOutputEOL,
	)

	fmt.Fprintf(
		&report,
		"%[1]s%[2]s%[1]s%[1]s",
		nagios.CheckOutputEOL,
		strings.Repeat("-", 50),
	)

	switch {

	case rcr.RebootRequired():
		fmt.Fprintf(
			&report,
			"Reboot required because:%[1]s%[1]s",
			nagios.CheckOutputEOL,
		)

		for _, result := range rcr {

			// TODO: Use panic here to surface the issue explicitly?
			if len(result.RebootReasons) > 0 && !result.RebootRequired {
				warningMessage := fmt.Sprintf(
					"BUG: RebootReasons (%d) recorded for %q, but RebootRequired flag not set",
					len(result.RebootReasons),
					result.Examined,
				)
				logger.Print(warningMessage)

				fmt.Fprint(
					&report,
					warningMessage,
					nagios.CheckOutputEOL,
				)
			}

			// If there is one or more reasons listed for why a reboot is
			// required that should be enough to indicate that the need for a
			// reboot was determined.
			// if len(result.RebootReasons) > 0 {
			//
			// Even so, perhaps there is an advantage to being overly
			// cautious?
			if len(result.RebootReasons) > 0 && result.RebootRequired {

				for _, reason := range result.RebootReasons {

					fmt.Fprintf(
						&report,
						"\n%3s %s%s",
						"-",
						reason,
						nagios.CheckOutputEOL,
					)

					if verbose {
						switch v := result.Examined.(type) {
						case restart.RebootRequiredAsserterWithDataDisplay:
							logger.Printf("Type assertion worked, value available for check result")

							fmt.Fprintf(
								&report,
								"    %s%s",
								v.DataDisplay(),
								nagios.CheckOutputEOL,
							)

						default:
							logger.Printf("Type assertion failed, value not available for check result")
							logger.Printf("Type found: %T", v)
						}
					}

				}
			}
		}

		fmt.Fprint(&report, nagios.CheckOutputEOL)

	case rcr.IsOKState():
		fmt.Fprintf(&report, "Reboot not required%s", nagios.CheckOutputEOL)

	}

	return report.String()

}
