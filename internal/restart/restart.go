// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package restart

import (
	"errors"

	"github.com/atc0005/go-nagios"
)

// ErrRebootRequired indicates that sufficient confidence in reboot assertions
// has been met and a reboot is needed.
var ErrRebootRequired = errors.New("reboot assertions matched, reboot needed")

// ErrMissingValue indicates that an expected value was missing.
var ErrMissingValue = errors.New("missing expected value")

// ErrUnknownRebootEvidence indicates that an unknown RebootEvidence indicator
// was specified.
var ErrUnknownRebootEvidence = errors.New("unknown RebootEvidence indicator")

// ErrInvalidRebootEvidence indicates that an invalid RebootEvidence indicator
// was specified.
var ErrInvalidRebootEvidence = errors.New("invalid RebootEvidence indicator")

// ErrMissingRequiredItem indicates that a required item (e.g., registry
// key, file) was not found and is required to be present.
var ErrMissingRequiredItem = errors.New("missing required reboot asserter")

// ErrMissingOptionalItem indicates that an optional item (e.g., registry
// key, file) was not found, though it is not required to be present.
var ErrMissingOptionalItem = errors.New("missing optional reboot asserter")

// RebootCheckResult is returned from an individual evaluation and indicates
// whether a reboot is required.
type RebootCheckResult struct {

	// Examined is the specific File or Key that was evaluated in order to
	// provide this evaluation/check result.
	//
	// NOTE: In most cases the enclosing type and not the embedded type (e.g.,
	// KeyInt instead of the embedded Key) should be recorded so that client
	// code will have access to additional fields (e.g., for use in final
	// report output).
	Examined RebootRequiredAsserter

	// Err records any error that occurs while performing an evaluation.
	Err error

	// RebootRequired provides the definitive decision regarding whether a
	// reboot is needed. In case of an error we assume that no reboot is
	// needed.
	RebootRequired bool // TODO: Make this a pointer?

	// RebootReasons is an optional human readable list of reasons why a
	// reboot is required.
	RebootReasons []string
}

// RebootCheckResults is a collection of "reboot required" evaluations and
// indicates whether a reboot is required.
type RebootCheckResults []RebootCheckResult

// ServiceStater represents a type that is capable of evaluating its overall
// state.
type ServiceStater interface {
	IsCriticalState() bool
	IsWarningState() bool
	IsOKState() bool
}

// RebootRequiredAsserter represents an item (reg key, file) that (if all
// requirements are matched) indicates the need for a reboot.
type RebootRequiredAsserter interface {
	Validate() error
	Evaluate() RebootCheckResult
	String() string
}

// RebootRequiredAsserterWithDataDisplay represents an item (reg key, file)
// that (if all requirements are matched) indicates the need for a reboot and
// provides the value associated with the item for display purposes.
type RebootRequiredAsserterWithDataDisplay interface {
	RebootRequiredAsserter

	// DataDisplay provides a string representation of a registry key's actual
	// data for display purposes.
	DataDisplay() string
}

// RebootRequiredAsserters is a collection of items that if (if all
// requirements are matched) indicate the need for a reboot.
type RebootRequiredAsserters []RebootRequiredAsserter

// Validate performs basic validation of all items in the collection. An error
// is returned for any validation failures.
func (rras RebootRequiredAsserters) Validate() error {
	logger.Printf("%d assertions to validate", len(rras))

	for _, rra := range rras {
		logger.Printf("Validating %s", rra.String())

		if err := rra.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// HasErrors indicates whether any of the evaluation results in the collection
// have an associated error.
func (rcr RebootCheckResults) HasErrors() bool {
	for _, result := range rcr {
		if result.Err != nil {
			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(result.Err, ErrMissingOptionalItem) {
				continue
			}

			return true
		}
	}

	return false

}

// NumErrors indicates how many of the evaluation results in the collection
// have an associated error.
func (rcr RebootCheckResults) NumErrors() int {
	var counter int

	for _, result := range rcr {
		if result.Err != nil {

			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(result.Err, ErrMissingOptionalItem) {
				continue
			}

			counter++
		}
	}

	return counter

}

// Errs returns a slice of all errors associated with evaluation results in
// the collection *EXCEPT* for unsuccessful optional assertions. An empty
// slice is returned if there are no errors.
func (rcr RebootCheckResults) Errs() []error {
	errs := make([]error, 0, rcr.NumErrors())
	for _, result := range rcr {
		if result.Err != nil {

			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(result.Err, ErrMissingOptionalItem) {
				continue
			}

			errs = append(errs, result.Err)
		}
	}

	return errs
}

// RebootRequired indicates whether any of the evaluation results in the
// collection indicate the need for a reboot.
func (rcr RebootCheckResults) RebootRequired() bool {
	for _, result := range rcr {
		if result.RebootRequired {
			return true
		}
	}

	return false
}

// RebootAssertionsApplied indicates how many reboot assertions were applied
// to build the evaluation results collection.
func (rcr RebootCheckResults) RebootAssertionsApplied() int {
	return len(rcr)
}

// RebootAssertionsMatched indicates how many of the evaluation results in the
// collection indicate the need for a reboot.
func (rcr RebootCheckResults) RebootAssertionsMatched() int {
	var counter int
	for _, result := range rcr {
		if result.RebootRequired {
			counter++
		}
	}

	return counter
}

// RebootAssertionsNotMatched indicates how many of the evaluation results in
// the collection do not indicate the need for a reboot.
func (rcr RebootCheckResults) RebootAssertionsNotMatched() int {
	var counter int
	for _, result := range rcr {
		if !result.RebootRequired {
			counter++
		}
	}

	return counter
}

// ServiceState returns the appropriate Service Check Status label and exit
// code for the collection's evaluation results.
func (rcr RebootCheckResults) ServiceState() nagios.ServiceState {
	var stateLabel string
	var stateExitCode int

	switch {
	case rcr.IsCriticalState():
		stateLabel = nagios.StateCRITICALLabel
		stateExitCode = nagios.StateCRITICALExitCode
	case rcr.IsWarningState():
		stateLabel = nagios.StateWARNINGLabel
		stateExitCode = nagios.StateWARNINGExitCode
	case rcr.IsOKState():
		stateLabel = nagios.StateOKLabel
		stateExitCode = nagios.StateOKExitCode
	default:
		stateLabel = nagios.StateUNKNOWNLabel
		stateExitCode = nagios.StateUNKNOWNExitCode
	}

	return nagios.ServiceState{
		Label:    stateLabel,
		ExitCode: stateExitCode,
	}
}

// IsCriticalState indicates whether any results in the collection have a
// CRITICAL state.
func (rcr RebootCheckResults) IsCriticalState() bool {
	switch {

	// If we could determine that a reboot is required we consider that to be
	// a WARNING state.
	case rcr.RebootRequired():
		return false

	// If we were unable to determine whether a reboot is required due to
	// errors we consider that to be a CRITICAL state.
	case rcr.HasErrors():
		return true

	// No reboot required and no errors, not CRITICAL state.
	default:
		return false

	}
}

// IsWarningState indicates whether any results in the collection have a
// WARNING state.
func (rcr RebootCheckResults) IsWarningState() bool {
	return rcr.RebootRequired()
}

// IsOKState indicates whether all results in the collection have an OK state.
func (rcr RebootCheckResults) IsOKState() bool {
	return !rcr.RebootRequired() && !rcr.HasErrors()
}
