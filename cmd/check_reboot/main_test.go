package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/atc0005/go-nagios"
)

// TestEmptyClientPerfDataAndConstructedExitStateProducesDefaultTimeMetric
// asserts that omitted performance data from client code produces a default
// time metric when using the ExitState constructor.
func TestEmptyClientPerfDataAndConstructedExitStateProducesDefaultTimeMetric(t *testing.T) {
	t.Parallel()

	// Setup ExitState type the same way that client code using the
	// constructor would.
	nagiosExitState := nagios.New()

	// Performance Data metrics are not emitted if we do not supply a
	// ServiceOutput value.
	nagiosExitState.ServiceOutput = "TacoTuesday"

	var outputBuffer strings.Builder

	nagiosExitState.SetOutputTarget(&outputBuffer)

	// os.Exit calls break tests
	nagiosExitState.SkipOSExit()

	// Process exit state, emit output to our output buffer.
	nagiosExitState.ReturnCheckResults()

	want := fmt.Sprintf(
		"%s | %s",
		nagiosExitState.ServiceOutput,
		"'time'=",
	)

	got := outputBuffer.String()

	if !strings.Contains(got, want) {
		t.Errorf("ERROR: Plugin output does not contain the expected time metric")
		t.Errorf("\nwant %q\ngot %q", want, got)
	} else {
		t.Logf("OK: Emitted performance data contains the expected time metric.")
	}
}
