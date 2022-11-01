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

// ServiceStater represents a type that is capable of evaluating its overall
// state.
type ServiceStater interface {
	IsCriticalState() bool
	IsWarningState() bool
	IsOKState() bool
}

// MatchedPath represents a the path to an item (e.g., reg key, file) that was
// successfully matched when evaluating an assertion.
type MatchedPath interface {
	Root() string
	Rel() string
	Base() string
	Full() string
	String() string
}

// RebootRequiredAsserter represents an item (reg key, file) that is able to
// determine the need for a reboot.
type RebootRequiredAsserter interface {

	// Values implementing this interface are able to determine their service
	// state.
	ServiceStater

	// Err does not apply filtering. This is left to higher-level code
	// operating on values implementing this interface.
	Err() error

	Validate() error
	Evaluate()
	String() string
	RebootReasons() []string
	Ignored() bool
	MatchedPaths() MatchedPaths
	RebootRequired() bool
	HasEvidence() bool
	Filter(ignorePatterns []string)
}

// RebootRequiredAsserterWithDataDisplay represents an item (reg key, file)
// that is able to determine the need for a reboot and provide the value
// associated with the item for display purposes.
type RebootRequiredAsserterWithDataDisplay interface {
	RebootRequiredAsserter

	// DataDisplay provides a string representation of an item's actual data
	// for display purposes.
	DataDisplay() string
}

// RebootRequiredAsserterWithSubPaths represents an item (reg key, file) that
// is able to determine the need for a reboot and if there is evidence of
// subpath matches.
type RebootRequiredAsserterWithSubPaths interface {
	RebootRequiredAsserter

	// HasSubPathMatches indicates whether an item has evidence of subpath
	// matches.
	HasSubPathMatches() bool
}

// RebootRequiredAsserters is a collection of items that if (if all
// requirements are matched) indicate the need for a reboot.
type RebootRequiredAsserters []RebootRequiredAsserter

// MatchedPaths is a collection of MatchedPath values.
type MatchedPaths []MatchedPath

// Validate performs basic validation of all items in the collection. An error
// is returned for any validation failures.
func (rras RebootRequiredAsserters) Validate() error {
	logger.Printf("%d assertions to validate", len(rras))

	for _, rra := range rras {
		logger.Printf("Validating %q", rra.String())

		if err := rra.Validate(); err != nil {
			return err
		}

		logger.Printf("Successfully validated %q", rra.String())
	}

	return nil
}

// Evaluate performs an evaluation of each assertion in the collection to
// determine whether a reboot is needed.
func (rras RebootRequiredAsserters) Evaluate() {
	for i := range rras {
		rras[i].Evaluate()
	}
}

// HasErrors indicates whether any of the assertion evaluations resulted in an
// error. Missing optional items are excluded. A boolean value is accepted
// which indicates whether assertion values marked as ignored (during
// filtering) should also be considered. The caller is responsible for
// filtering the collection prior to calling this method.
func (rras RebootRequiredAsserters) HasErrors(evalIgnored bool) bool {
	for _, assertion := range rras {
		if assertion.Err() != nil {
			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(assertion.Err(), ErrMissingOptionalItem) {
				continue
			}

			if assertion.Ignored() && !evalIgnored {
				continue
			}

			return true
		}
	}

	return false

}

// NumErrors indicates how many of the evaluation results in the collection
// have an associated error. Missing optional items are excluded. A boolean
// value is accepted which indicates whether assertion values marked as
// ignored (during filtering) should also be considered. The caller is
// responsible for filtering the collection prior to calling this method.
func (rras RebootRequiredAsserters) NumErrors(evalIgnored bool) int {
	var counter int

	for _, assertion := range rras {
		if assertion.Err() != nil {

			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(assertion.Err(), ErrMissingOptionalItem) {
				continue
			}

			if assertion.Ignored() && !evalIgnored {
				continue
			}

			counter++
		}
	}

	return counter
}

// Errs returns a slice of all errors associated with evaluation results in
// the collection. Missing optional items are excluded. A boolean value is
// accepted which indicates whether assertion values marked as ignored (during
// filtering) should also be considered.
//
// The caller is responsible for filtering the collection prior to calling
// this method.
func (rras RebootRequiredAsserters) Errs(evalIgnored bool) []error {
	errs := make([]error, 0, rras.NumErrors(evalIgnored))
	for _, assertion := range rras {
		if assertion.Err() != nil {

			// Don't report unsuccessful matches for optional items.
			//
			// TODO: Should we use this method to surface all errors and
			// provide another method to filter out optional item match
			// errors?
			if errors.Is(assertion.Err(), ErrMissingOptionalItem) {
				continue
			}

			if assertion.Ignored() && !evalIgnored {
				continue
			}

			errs = append(errs, assertion.Err())
		}
	}

	return errs
}

// RebootRequired indicates whether any of the evaluation results in the
// collection indicate the need for a reboot.
func (rras RebootRequiredAsserters) RebootRequired() bool {
	for _, assertion := range rras {

		// NOTE: We're relying on the assertion-specific logic to filter out
		// any that have been marked as ignored.
		if assertion.RebootRequired() {
			return true
		}
	}

	return false
}

// NumApplied indicates how many reboot assertions were applied.
func (rras RebootRequiredAsserters) NumApplied() int {
	return len(rras)
}

// NumMatched indicates how many of the items in the collection indicate the
// need for a reboot. The caller is responsible for filtering the collection
// prior to calling this method.
func (rras RebootRequiredAsserters) NumMatched() int {
	var counter int
	for _, assertion := range rras {
		if assertion.RebootRequired() {
			counter++
		}
	}

	return counter
}

// NumNotMatched indicates how many of the items in the collection do not
// indicate the need for a reboot. The caller is responsible for filtering the
// collection prior to calling this method.
func (rras RebootRequiredAsserters) NumNotMatched() int {
	var counter int
	for _, assertion := range rras {
		if !assertion.RebootRequired() {
			counter++
		}
	}

	return counter
}

// ServiceState returns the appropriate Service Check Status label and exit
// code for the collection's evaluation results. The caller is responsible for
// filtering the collection prior to calling this method.
func (rras RebootRequiredAsserters) ServiceState() nagios.ServiceState {
	var stateLabel string
	var stateExitCode int

	switch {
	case rras.HasCriticalState():
		stateLabel = nagios.StateCRITICALLabel
		stateExitCode = nagios.StateCRITICALExitCode
	case rras.HasWarningState():
		stateLabel = nagios.StateWARNINGLabel
		stateExitCode = nagios.StateWARNINGExitCode
	case rras.IsOKState():
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

// HasCriticalState indicates whether any items in the collection were
// evaluated to a CRITICAL state. The caller is responsible for filtering the
// collection prior to calling this method.
func (rras RebootRequiredAsserters) HasCriticalState() bool {
	for _, assertion := range rras {
		if assertion.IsCriticalState() {
			return true
		}
	}

	return false
}

// HasWarningState indicates whether any items in the collection were
// evaluated to a WARNING state. The caller is responsible for filtering the
// collection prior to calling this method.
func (rras RebootRequiredAsserters) HasWarningState() bool {
	for _, assertion := range rras {
		if assertion.IsWarningState() {
			return true
		}
	}

	return false
}

// IsOKState indicates whether all items in the collection were evaluated to
// an OK state. The caller is responsible for filtering the collection prior
// to calling this method.
func (rras RebootRequiredAsserters) IsOKState() bool {
	for _, assertion := range rras {
		if !assertion.IsOKState() {
			logger.Printf("%q failed IsOKState() check", assertion.String())
			return false
		}
	}

	return true
}

// HasRebootRequired indicates whether the collection contains an entry whose
// evaluation indicates that a reboot is needed. Entries marked as ignored are
// not considered. The caller is responsible for filtering the collection
// prior to calling this method.
func (rras RebootRequiredAsserters) HasRebootRequired() bool {
	for _, assertion := range rras {
		if assertion.RebootRequired() {
			return true
		}
	}

	return false
}

// HasIgnored indicates whether the collection contains an entry marked as
// ignored. The caller is responsible for filtering the collection prior to
// calling this method.
func (rras RebootRequiredAsserters) HasIgnored() bool {
	for _, assertion := range rras {
		if assertion.Ignored() {
			return true
		}
	}

	return false
}

// Filter uses the list of specified ignore patterns to mark any applicable
// items in the collection as ignored. Filter should be called before
// performing final state evaluation.
func (rras RebootRequiredAsserters) Filter(ignorePatterns []string) {
	for i := range rras {
		rras[i].Filter(ignorePatterns)
	}
}

// NumIgnored indicates how many of the items in the collection have been
// marked as ignored. The caller is responsible for filtering the collection
// prior to calling this method.
func (rras RebootRequiredAsserters) NumIgnored() int {
	var counter int
	for _, assertion := range rras {
		if assertion.Ignored() {
			logger.Printf("%q MARKED AS IGNORED", assertion)
			counter++
		}
	}

	return counter
}

// NumNotIgnored indicates how many of the items in the collection have not
// been marked as ignored. The caller is responsible for filtering the
// collection prior to calling this method.
func (rras RebootRequiredAsserters) NumNotIgnored() int {
	var counter int
	for _, assertion := range rras {
		if !assertion.Ignored() {
			counter++
		}
	}

	return counter
}

// NotIgnoredItems returns all items in the collection that have not been
// marked as ignored. If all items have been marked as ignored an empty
// collection is returned. The caller is responsible for filtering the
// collection prior to calling this method.
func (rras RebootRequiredAsserters) NotIgnoredItems() RebootRequiredAsserters {
	logger.Printf("%d items not marked as ignored", rras.NumNotIgnored())

	assertions := make(RebootRequiredAsserters, 0, rras.NumNotIgnored())
	for _, assertion := range rras {
		if !assertion.Ignored() {
			assertions = append(assertions, assertion)
		}
	}

	return assertions
}

// IgnoredItems returns all items in the collection that have been marked as
// ignored. If no items have been marked as ignored an empty collection is
// returned. The caller is responsible for filtering the collection prior to
// calling this method.
func (rras RebootRequiredAsserters) IgnoredItems() RebootRequiredAsserters {
	logger.Printf("%d items marked as ignored", rras.NumIgnored())

	assertions := make(RebootRequiredAsserters, 0, rras.NumIgnored())
	for _, assertion := range rras {
		if assertion.Ignored() {
			assertions = append(assertions, assertion)
		}
	}

	logger.Printf("Returning %d items marked as ignored", len(assertions))

	return assertions
}
