//go:build windows

// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package registry

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/check-restart/internal/textutils"
	"golang.org/x/sys/windows/registry"
)

// Add "implements assertions" to fail the build if the
// restart.RebootRequiredAsserter implementation isn't correct.
var (
	_ restart.RebootRequiredAsserter = (*Key)(nil)
	_ restart.RebootRequiredAsserter = (*KeyBinary)(nil)
	_ restart.RebootRequiredAsserter = (*KeyInt)(nil)
	_ restart.RebootRequiredAsserter = (*KeyString)(nil)
	_ restart.RebootRequiredAsserter = (*KeyStrings)(nil)
	_ restart.RebootRequiredAsserter = (*KeyPair)(nil)
)

// Add "implements assertions" to fail the build if the
// restart.RebootRequiredAsserterWithDataDisplay implementation isn't correct.
var (
	_ restart.RebootRequiredAsserterWithDataDisplay = (*KeyBinary)(nil)
	_ restart.RebootRequiredAsserterWithDataDisplay = (*KeyInt)(nil)
	_ restart.RebootRequiredAsserterWithDataDisplay = (*KeyString)(nil)
	_ restart.RebootRequiredAsserterWithDataDisplay = (*KeyStrings)(nil)
	_ restart.RebootRequiredAsserterWithDataDisplay = (*KeyPair)(nil)
)

// Add "implements assertions" to fail the build if the
// restart.RebootRequiredAsserterWithSubPaths implementation isn't correct.
var (
	_ restart.RebootRequiredAsserterWithSubPaths = (*KeyBinary)(nil)
	_ restart.RebootRequiredAsserterWithSubPaths = (*KeyInt)(nil)
	_ restart.RebootRequiredAsserterWithSubPaths = (*KeyString)(nil)
	_ restart.RebootRequiredAsserterWithSubPaths = (*KeyStrings)(nil)
)

// Add "implements assertions" to fail the build if the restart.MatchedPath
// implementation isn't correct.
var _ restart.MatchedPath = (*MatchedPath)(nil)

// ErrUnsupportedOS indicates that an unsupported OS has been detected.
var ErrUnsupportedOS = errors.New("unsupported OS detected; this package requires a Windows OS to run properly")

// ErrInvalidNumberOfKeysInKeyPair indicates that either too few or too many
// keys were provided for a key pair.
var ErrInvalidNumberOfKeysInKeyPair = errors.New("invalid number of keys in key pair")

// ErrInvalidRootKey indicates that an invalid registry root key was
// specified.
var ErrInvalidRootKey = errors.New("invalid root key")

// ErrMissingKey indicates that a requested registry key is missing.
// var ErrMissingKey = errors.New("missing expected key")

// ErrMissingRequiredKey indicates that a requested (and required) registry
// key is missing.
var ErrMissingRequiredKey = errors.New("missing required key")

// ErrMissingOptionalKey indicates that a requested (and optional) registry
// key is missing.
var ErrMissingOptionalKey = errors.New("missing optional key")

// ErrKeyAlreadyOpen indicates that a specified registry key is already open.
// This error indicates that there is likely a logic bug somewhere in the
// caller's code.
var ErrKeyAlreadyOpen = errors.New("registry key is already open")

// ErrKeyNotOpen indicates that a registry key is not open. This error
// indicates that there is likely a logic bug somewhere in the caller's code.
var ErrKeyNotOpen = errors.New("registry key is not open")

// Registry value types.
// https://pkg.go.dev/golang.org/x/sys/windows/registry#pkg-constants
const (
	RegKeyTypeNone                     = "NONE"
	RegKeyTypeSZ                       = "SZ"
	RegKeyTypeExpandSZ                 = "EXPAND_SZ"
	RegKeyTypeBinary                   = "BINARY"
	RegKeyTypeDWORD                    = "DWORD"
	RegKeyTypeDWORDBigEndian           = "DWORD_BIG_ENDIAN"
	RegKeyTypeLink                     = "LINK"
	RegKeyTypeMultiSZ                  = "MULTI_SZ"
	RegKeyTypeResourceList             = "RESOURCE_LIST"
	RegKeyTypeFullResourceDescriptor   = "FULL_RESOURCE_DESCRIPTOR"
	RegKeyTypeResourceRequirementsList = "RESOURCE_REQUIREMENTS_LIST"
	RegKeyTypeQWORD                    = "QWORD"
	RegKeyTypeUnknown                  = "UNKNOWN" // fallback value
)

// Registry "root" key names.
const (
	RegKeyRootNameClassesRoot     = "HKEY_CLASSES_ROOT"
	RegKeyRootNameCurrentUser     = "HKEY_CURRENT_USER"
	RegKeyRootNameLocalMachine    = "HKEY_LOCAL_MACHINE"
	RegKeyRootNameUsers           = "HKEY_USERS"
	RegKeyRootNameCurrentConfig   = "HKEY_CURRENT_CONFIG"
	RegKeyRootNamePerformanceData = "HKEY_PERFORMANCE_DATA"
	RegKeyRootNameUnknown         = "UNKNOWN" // fallback value
)

const (
	// pendingFileRenameOperationsPrefix is a prefix found for entries in the
	// REG_MULTI_SZ registry key value named PendingFileRenameOperations.
	pendingFileRenameOperationsPrefix string = `\??\`

	// RegKeyTypeMultiSZDataDisplayLimit is the limit or sampling size used
	// when generating a string representation of a multi-string registry key
	// value for display purposes.
	//
	// Due to issues encountered with NSClient++ truncating output we need to
	// keep this value small in order to reduce the chance that output from
	// other required required evidence is lost when emitting verbose details
	// for this registry key type.
	RegKeyTypeMultiSZDataDisplayLimit int = 2
)

// Key requirement labels used by logging and error messages to provide
// additional context to messages.
const (
	KeyReqOptionalLabel = "optional"
	KeyReqRequiredLabel = "required"
)

// KeyRebootRequired represents the behavior of a registry key that can be
// evaluated to indicate whether a reboot is required.
//
// NOTE: As of the v0.1.0 release this interface is not used, though a prior
// version of the client code did use this. Keeping around for the time being.
type KeyRebootRequired interface {
	Validate() error
	Evidence() KeyRebootEvidence
	Requirements() KeyAssertions
	Path() string
	Value() string
	RootKey() registry.Key
	String() string
}

// MatchedPath represents a path that was matched when performing an
// evaluation of a "reboot required" assertion.
type MatchedPath struct {
	// root is the left-most element of a matched path. This is the beginning
	// of a qualified path.
	root string

	// relative is the unqualified path (without the root element). The base
	// element of the path is usually included in this element.
	relative string

	// base is the last element or the right-most "leaf" value of a matched
	// path.
	base string

	// ignored indicates whether this value has been marked by filtering logic
	// as not considered when determining whether a reboot is needed.
	ignored bool
}

// MatchedPathIndex is a collection of path values that were matched during
// evaluation of specified reboot required assertions.
type MatchedPathIndex map[string]MatchedPath

// Root returns the left-most element of a matched path. This returned value
// is the beginning of a qualified path.
func (mp MatchedPath) Root() string {
	return mp.root
}

// Rel returns the relative (unqualified) element of a matched path. The base
// element of the path is usually included in this element.
func (mp MatchedPath) Rel() string {
	return mp.relative
}

// Base returns the last or right-most "leaf" element of a matched path.
func (mp MatchedPath) Base() string {
	return mp.base
}

// Full returns the qualified matched path value.
func (mp MatchedPath) Full() string {
	// return filepath.Join(mp.root, mp.relative)
	return fmt.Sprintf(`%v\%s`, mp.root, mp.relative)
}

// String provides a human readable version of the matched path value.
func (mp MatchedPath) String() string {
	return mp.Full()
}

// KeyRebootEvidence indicates what registry key evidence is required in order
// to determine that a reboot is needed.
type KeyRebootEvidence struct {
	// DataOtherThanX indicates that a registry key value data field with a
	// value other than the one indicated is sufficient evidence for a reboot.
	DataOtherThanX bool

	// SubKeysExist indicates that the existence of registry key subkeys is
	// sufficient evidence for a reboot.
	SubKeysExist bool

	// ValueExists indicates that the existence of a registry key value is
	// sufficient evidence for a reboot.
	ValueExists bool

	// KeyExists indicates that the existence of a registry key path is
	// sufficient evidence for a reboot.
	KeyExists bool
}

// KeyPairRebootEvidence applies additional evidence "markers" for the KeyPair
// type. If the reboot evidence markers for the enclosed Keys are not matched,
// this (also optional) evidence marker is then checked to determine if a
// reboot is required for the pair as a whole.
type KeyPairRebootEvidence struct {
	// PairedValuesDoNotMatch indicates that one registry key value data field
	// differing from a second registry key value data field is sufficient
	// evidence for a reboot.
	PairedValuesDoNotMatch bool
}

// KeyStringsRebootEvidence applies additional evidence "markers" for the
// KeyStrings type. If the reboot evidence markers for the Key type are not
// matched, these  (also optional) set of evidence markers are then checked to
// determine if a reboot is required.
type KeyStringsRebootEvidence struct {

	// ValueFound is an evidence "marker" that if satisfied indicates the need
	// for a reboot. This marker allows for any single value match.
	ValueFound bool

	// AllValuesFound is an evidence "marker" that if satisfied indicates the
	// need for a reboot. This is an "all or nothing" requirement; all
	// expected values much be found.
	AllValuesFound bool
}

// KeyAssertions indicates what requirements must be met. If not met, this
// indicates that an error has occurred. If a specific registry key or value
// is required, but not present on a system then client code can not reliably
// determine whether a reboot is necessary.
type KeyAssertions struct {

	// KeyRequired is optionally used to indicate that a registry key is
	// required.
	KeyRequired bool

	// ValueRequired is optionally used to indicate that a registry key value
	// is required.
	ValueRequired bool
}

// KeyRuntime is a collection of values for a Key that are set during Key
// evaluation. Unlike the static values set for a Key (e.g., root key, path,
// any requirements or key assertions), these values are not known until
// execution or runtime.
type KeyRuntime struct {

	// handle is a handle to an open registry key. This is a required value
	// used by evaluation logic applied by the "base" Key and "super" types
	// enclosing it.
	//
	// Per official documentation, a handle to an open registry key should not
	// be used after it is closed and should not remain open any longer than
	// necessary.
	// https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regclosekey
	handle *registry.Key

	// err records any error that occurs while performing an evaluation.
	err error

	// evidenceFound is the collection of evidence found when evaluating
	// a specified assertion.
	evidenceFound KeyRebootEvidence

	// ignored indicates whether this value has been marked by filtering logic
	// as not considered when determining whether a reboot is needed.
	// ignored bool

	// valueType represents the type of a specified registry value. This field
	// is only set when a value is specified for a registry key assertion.
	valueType string

	// pathsMatched is a collection of path values that were matched during
	// evaluation of specified reboot required assertions.
	pathsMatched MatchedPathIndex
}

// Key represents a registry key that if found (and requirements met)
// indicates a reboot is needed.
type Key struct {
	// root is the root or base registry key (e.g, HKEY_LOCAL_MACHINE).
	root registry.Key

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyRuntime

	// path is the registry key path minus the root key (e.g.,
	// HKEY_LOCAL_MACHINE) and any value to be evaluated.
	path string

	// value is the registry key value name.
	value string

	// evidenceExpected indicates what evidence is used to determine that a
	// reboot is needed.
	evidenceExpected KeyRebootEvidence

	// requirements indicates what requirements must be met. If not met, this
	// indicates that an error has occurred.
	requirements KeyAssertions
}

// Keys is a collection of Key values.
type Keys []*Key

// KeysRebootRequired is a collection of KeysRebootRequired values.
type KeysRebootRequired []KeyRebootRequired

// KeyPairRuntime is a collection of values that are set during evaluation.
// Unlike static values that are known ahead of time, these values are not
// known until execution or runtime.
type KeyPairRuntime struct {
	// data represents the data stored for both registry key values.
	data [][]byte

	// evidenceFound is the collection of evidence found when evaluating
	// a specified assertion.
	evidenceFound KeyPairRebootEvidence
}

// KeyPair represents two Keys that are evaluated together.
type KeyPair struct {
	Keys Keys

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyPairRuntime

	// additionalEvidence applies additional evidence "markers" for this type.
	// If the reboot evidence markers for the enclosed Key type are not
	// matched, this (also optional) set of evidence markers are then checked
	// to determine if a reboot is required.
	additionalEvidence KeyPairRebootEvidence
}

// KeyIntRuntime is a collection of values that are set during evaluation.
// Unlike static values that are known ahead of time, these values are not
// known until execution or runtime.
type KeyIntRuntime struct {
	// data represents the data stored for a registry key value.
	data uint64
}

// KeyInt represents a Key containing integer data for comparison.
type KeyInt struct {
	Key

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyIntRuntime

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData uint64
}

// KeyBinaryRuntime is a collection of values that are set during evaluation.
// Unlike static values that are known ahead of time, these values are not
// known until execution or runtime.
type KeyBinaryRuntime struct {
	// data represents the actual data stored for a registry key value.
	data []byte
}

// KeyBinary represents a Key containing binary data for comparison.
type KeyBinary struct {
	Key

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyBinaryRuntime

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData []byte
}

// KeyStringRuntime is a collection of values that are set during evaluation.
// Unlike static values that are known ahead of time, these values are not
// known until execution or runtime.
type KeyStringRuntime struct {
	// data represents the actual data stored for a registry key value.
	data string
}

// KeyString represents a Key containing string data for comparison.
type KeyString struct {
	Key

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyStringRuntime

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData string
}

// KeyStringsRuntime is a collection of values that are set during evaluation.
// Unlike static values that are known ahead of time, these values are not
// known until execution or runtime.
type KeyStringsRuntime struct {
	// data represents the actual data stored for a registry key value.
	data []string

	// searchTermMatched is the specific search term that was matched out of
	// the expected data (collection of strings). This is only set when the
	// expected evidence is a single value match and not when matching all
	// values.
	searchTermMatched string

	// evidenceFound is the collection of evidence found when evaluating
	// a specified assertion.
	evidenceFound KeyStringsRebootEvidence
}

// KeyStrings represents a Key containing multiple strings for comparison.
// That collection of strings maps to a registry.MULTI_SZ key type and is
// retrieved as a slice of strings.
type KeyStrings struct {
	Key

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime KeyStringsRuntime

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData []string

	// additionalEvidence applies additional evidence "markers" for this type.
	// If the reboot evidence markers for the enclosed Key type are not
	// matched, this (also optional) set of evidence markers are then checked
	// to determine if a reboot is required.
	additionalEvidence KeyStringsRebootEvidence
}

// AddMatchedPath records given paths as successful assertion matches.
// Duplicate entries are ignored.
func (k *Key) AddMatchedPath(paths ...string) {

	if k.runtime.pathsMatched == nil {
		k.runtime.pathsMatched = make(MatchedPathIndex)
	}

	for _, path := range paths {
		// Record MatchedPath if it does not already exist; we do not want to
		// overwrite an existing entry in case any non-default metadata is set
		// for the entry.
		if _, ok := k.runtime.pathsMatched[path]; !ok {
			matchedPath := MatchedPath{
				root:     getRootKeyName(k.RootKey()),
				relative: path,
				base:     filepath.Base(path),
			}

			k.runtime.pathsMatched[path] = matchedPath
		}
	}
}

// MatchedPaths returns all recorded paths from successful assertion matches.
// func (k *Key) MatchedPaths() []string {
// 	paths := make([]string, 0, len(k.runtime.pathsMatched))
//
// 	for path := range k.runtime.pathsMatched {
// 		paths = append(paths, path)
// 	}
//
// 	return sort.StringSlice(paths)
// }

// MatchedPaths returns all recorded paths from successful assertion
// matches.
func (k *Key) MatchedPaths() restart.MatchedPaths {

	pathStrings := make([]string, 0, len(k.runtime.pathsMatched))
	matchedPaths := make(restart.MatchedPaths, 0, len(k.runtime.pathsMatched))

	// Pull all of the keys.
	for k := range k.runtime.pathsMatched {
		pathStrings = append(pathStrings, k)
	}

	// Sort them.
	sort.Strings(sort.StringSlice(pathStrings))

	// Use them to pull out the MatchedPath entries in order.
	for _, path := range pathStrings {
		logger.Printf("Key.runtime.pathsMatched entry: %q", path)
		matchedPaths = append(matchedPaths, k.runtime.pathsMatched[path])
	}

	return matchedPaths
}

// SetFoundEvidenceKeyExists records that the KeyExists reboot evidence was found.
func (k *Key) SetFoundEvidenceKeyExists() {
	logger.Printf("Recording that the KeyExists evidence was found for %q", k)
	k.runtime.evidenceFound.KeyExists = true
}

// SetFoundEvidenceValueExists records that the ValueExists reboot evidence
// was found.
func (k *Key) SetFoundEvidenceValueExists() {
	logger.Printf("Recording that the ValueExists evidence was found for %q", k)
	k.runtime.evidenceFound.ValueExists = true
}

// SetFoundEvidenceSubKeysExist records that the SubKeysExist reboot evidence
// was found.
func (k *Key) SetFoundEvidenceSubKeysExist() {
	logger.Printf("Recording that the SubKeysExist evidence was found for %q", k)
	k.runtime.evidenceFound.SubKeysExist = true
}

// SetFoundEvidenceDataOtherThanX records that the DataOtherThanX reboot
// evidence was found.
func (k *Key) SetFoundEvidenceDataOtherThanX() {
	logger.Printf("Recording that the DataOtherThanX evidence was found for %q", k)
	k.runtime.evidenceFound.DataOtherThanX = true
}

// ExpectedEvidence returns the specified evidence that (if found) indicates a
// reboot is needed.
func (k *Key) ExpectedEvidence() KeyRebootEvidence {
	return k.evidenceExpected
}

// DiscoveredEvidence returns the discovered evidence from an earlier
// evaluation.
func (k *Key) DiscoveredEvidence() KeyRebootEvidence {
	return k.runtime.evidenceFound
}

// HasEvidence indicates whether any evidence was found for an assertion
// evaluation.
func (k *Key) HasEvidence() bool {
	if k.runtime.evidenceFound.DataOtherThanX {
		return true
	}

	if k.runtime.evidenceFound.KeyExists {
		return true
	}

	if k.runtime.evidenceFound.SubKeysExist {
		return true
	}

	if k.runtime.evidenceFound.ValueExists {
		return true
	}

	return false
}

// RebootReasons returns a list of the reasons associated with the evidence
// found for an evaluation that indicates a reboot is needed.
func (k *Key) RebootReasons() []string {

	// The usual scenario is one reason per evidence match.
	reasons := make([]string, 0, 1)

	if k.runtime.evidenceFound.DataOtherThanX {
		reasons = append(reasons, fmt.Sprintf(
			"Data for value %s for key %s found", k.Value(), k,
		))
	}

	if k.runtime.evidenceFound.KeyExists {
		reasons = append(reasons, fmt.Sprintf(
			"Key %s found", k,
		))
	}

	if k.runtime.evidenceFound.SubKeysExist {
		reasons = append(reasons, fmt.Sprintf(
			"Subkeys for key %s found", k,
		))
	}

	if k.runtime.evidenceFound.ValueExists {
		switch {
		case k.runtime.valueType != "":
			reasons = append(reasons, fmt.Sprintf(
				"Value %s of type %s for key %s found",
				k.Value(),
				k.runtime.valueType,
				k,
			))
		default:
			logger.Print(
				"BUG: k.runtime.valueType should have been recorded " +
					"when evaluating a specified registry key value",
			)
			reasons = append(reasons, fmt.Sprintf(
				"Value %s for key %s found",
				k.Value(),
				k,
			))
		}
	}

	return reasons
}

// String provides the fully qualified path for a Key.
func (k *Key) String() string {

	// NOTE: Printing this way does not indicate what registry key values were
	// checked.
	//
	// This is probably necessary due to how the Key value is referenced, but
	// will need to consider how to force displaying the registry key value
	// also.
	return fmt.Sprintf(
		`%v\%s`,
		getRootKeyName(k.root),
		k.path,
	)
}

// Requirements returns the specified requirements or key assertions. If one
// of these requirements is not met then an error condition has been
// encountered. Requirements does not indicate whether a reboot is needed,
// only how potential "key not found" or "value not found" conditions should
// be treated.
func (k *Key) Requirements() KeyAssertions {
	return k.requirements
}

// Path returns the specified (unqualified) registry key path.
func (k *Key) Path() string {
	return k.path
}

// RootKey returns the specified registry root key.
func (k *Key) RootKey() registry.Key {
	return k.root
}

// Value returns the specified registry key value.
func (k *Key) Value() string {
	return k.value
}

// Handle returns the current handle to the open registry key if it exists,
// otherwise returns nil.
func (k *Key) Handle() *registry.Key {
	return k.runtime.handle
}

// open creates a handle to the registry key and saves it for later use. The
// caller is responsible for calling the Close method to free the resources
// used by the open registry key.
func (k *Key) open() error {
	// Skip opening a handle to the registry key if it is already open.
	if k.runtime.handle != nil {
		logger.Printf("Handle exists; registry key %q is already open", k)
		return ErrKeyAlreadyOpen
	}

	logger.Printf("Handle does not exist, attempting to open registry key %q", k)

	// Enumerating subkeys requires requesting access to do so along with
	// permission to query values.
	//
	// We specify both permissions by combining the values via OR.
	// https://stackoverflow.com/questions/47814070/golang-cant-enumerate-subkeys-of-registry-key
	openKey, err := registry.OpenKey(k.RootKey(), k.Path(), registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if k.Requirements().KeyRequired {
			logger.Printf("Key %q not found, but marked as required.", k)
			return ErrMissingRequiredKey
		}

		logger.Printf("Key %q not found, but not marked as required.", k)

		// TODO: Perhaps return nil instead? What do we really gain by
		// returning an error that isn't intended to be actionable?
		return ErrMissingOptionalKey

	case err != nil:
		keyReqLabel := KeyReqOptionalLabel
		if k.Requirements().KeyRequired {
			keyReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while opening %s key %q: %s",
			keyReqLabel,
			k,
			err,
		)

		return fmt.Errorf(
			"unexpected error occurred while opening %s key %s: %w",
			keyReqLabel,
			k,
			err,
		)

	}

	k.runtime.handle = &openKey

	// TODO: Any other feasible way to handle this? This is a logic problem
	// that needs to be resolved.
	if k.runtime.handle == nil {
		panic("BUG: k.runtime.handle is nil and should not be. Explosions commence!")
	}

	return nil

}

// closeAndLog wraps the handle closure logic with additional logging for
// troubleshooting purposes.
func (k *Key) closeAndLog() {
	logger.Printf("Attempting to close handle to %q", k)
	if err := k.close(); err != nil {
		logger.Printf("Failed to close handle to open key %q", k)
	}

	if k.Handle() == nil {
		logger.Printf("Handle to %q closed", k)
	}
}

// close will close the handle to a registry key if open, otherwise will
// act as a NOOP. An error is returned if one is encountered when attempting
// to close the handle.
func (k *Key) close() error {

	if k.runtime.handle == nil {
		logger.Printf("Handle for %s already closed", k)
		return nil
	}

	if err := k.runtime.handle.Close(); err != nil {
		logger.Printf("Error encountered closing handle to %s: %v", k, err)
		return err
	}

	// Remove reference to the handle since it is no longer valid.
	logger.Printf("Removed reference to the handle since it is no longer valid")
	k.runtime.handle = nil

	return nil

}

// Validate performs basic validation. An error is returned for any validation
// failures.
func (k *Key) Validate() error {

	switch getRootKeyName(k.root) {
	case RegKeyRootNameUnknown:
		return fmt.Errorf(
			"registry root key unknown: %w",
			ErrInvalidRootKey,
		)
	default:
		// OK scenario
	}

	if k.path == "" {
		return fmt.Errorf(
			"required registry key path not specified: %w",
			restart.ErrMissingValue,
		)
	}

	// Having an empty Value is acceptable only for assertions which do not
	// require it. For example, if we are only looking for the presence of the
	// key or subkeys we do not need the key value.
	if k.Value() == "" && k.evidenceExpected.ValueExists {
		// logger.Printf("evidence: %+v", k.evidence)
		return fmt.Errorf(
			"required registry value not specified: %w",
			restart.ErrMissingValue,
		)
	}

	// Validate reboot evidence values.
	switch {
	case k.evidenceExpected.DataOtherThanX:
	case k.evidenceExpected.SubKeysExist:
	case k.evidenceExpected.ValueExists:
	case k.evidenceExpected.KeyExists:
	default:

		// For all cases other than KeyPair types one of the reboot evidence
		// fields should be set to true.
		//
		// For KeyPair types each Key in the Keys collection will have all
		// reboot evidence fields set to false and the key assertion fields
		// for key and value set to true to indicate that both are required.
		//
		// Additionally, the KeyPair type has a separate reboot evidence field
		// that indicates we are looking for paired values that do not match
		// to indicate a reboot.
		// if !(k.requirements.KeyRequired && k.requirements.ValueRequired) {
		if !k.requirements.KeyRequired || !k.requirements.ValueRequired {
			return fmt.Errorf(
				"value unexpected: %w",
				restart.ErrUnknownRebootEvidence,
			)
		}
	}

	return nil

}

// evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
//
// Depending on the value provided, a handle to an open registry key is
// retained after execution completes so that a "super type" key can perform
// further evaluation of registry key data.
func (k *Key) evaluate(closeHandle bool) {
	logger.Printf("Evaluating key %q", k)

	if err := k.evalOpenKey(); err != nil {
		logger.Print("Evaluation of specified registry key unsuccessful")

		// Replace with general error value that the client code can more
		// easily use to determine severity.
		switch {
		case errors.Is(err, ErrMissingOptionalKey):
			logger.Printf("evalOpenKey(): Setting ErrMissingOptionalKey for %q", k)
			k.runtime.err = restart.ErrMissingOptionalItem

		case errors.Is(err, ErrMissingRequiredKey):
			logger.Printf("evalOpenKey(): Setting ErrMissingRequiredKey for %q", k)
			k.runtime.err = restart.ErrMissingRequiredItem
		default:
			logger.Printf("evalOpenKey(): Setting general error for %q", k)
			k.runtime.err = err
		}

		return
	}

	// Only attempt to close the handle if we successfully opened it and if we
	// were asked to close it.
	defer func() {
		if !closeHandle {
			logger.Printf("Skipping closure of handle to %q as requested", k)
			return
		}

		logger.Printf("Attempting to close handle to %q", k)
		if err := k.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", k)
		}
	}()

	// If evidence of the need for a reboot is found skip any further checks.
	if k.HasEvidence() {
		logger.Printf("HasEvidence() early exit triggered %q", k)
		return
	}

	if err := k.evalValue(); err != nil {
		logger.Printf("evalValue() error for %q: %s", k, err)
		k.runtime.err = err
		return
	}

	if err := k.evalSubKeys(); err != nil {
		logger.Printf("evalSubKeys() error for %q: %s", k, err)
		k.runtime.err = err
		return
	}
}

// evalOpenKey performs the tasks needed to open a handle to the registry key
// and evaluate whether there is a need for a reboot. The handle to the open
// registry key is retained for later use.
//
// The caller is responsible for calling the close method to free resources
// used by the open registry key.
func (k *Key) evalOpenKey() error {

	logger.Printf("Opening key %q", k)

	err := k.open()
	switch {
	case errors.Is(err, ErrKeyAlreadyOpen):
		logger.Printf("Key %q is already open?", k)
		logger.Print("TODO: Probably worth checking how this occurred.")

		return fmt.Errorf(
			"evalOpenKey() for key %s failed: %w", k, ErrKeyAlreadyOpen,
		)

	case errors.Is(err, ErrMissingRequiredKey):
		logger.Printf("Key %q not found, but marked as required.", k)
		return fmt.Errorf(
			"evalOpenKey() for key %s failed: %w", k, ErrMissingRequiredKey,
		)

	case errors.Is(err, ErrMissingOptionalKey):
		logger.Printf("Key %q not found, but not marked as required.", k)
		return fmt.Errorf(
			"evalOpenKey() for key %s unsuccessful: %w", k, ErrMissingOptionalKey,
		)

	case err != nil:
		keyReqLabel := KeyReqOptionalLabel
		if k.Requirements().KeyRequired {
			keyReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while opening %s key %q: %s",
			keyReqLabel,
			k,
			err,
		)

		return fmt.Errorf(
			"evalOpenKey() for key %s failed: %s: %w",
			keyReqLabel,
			k,
			err,
		)

	default:

		logger.Printf("Key %q opened ...", k)

		if k.ExpectedEvidence().KeyExists {
			logger.Println("Reboot Evidence found!")
			k.SetFoundEvidenceKeyExists()
			k.AddMatchedPath(k.Path())
		}

	}

	return nil
}

// evalSubKeys performs the tasks needed to evaluate whether the presence of
// subkeys for a given registry key indicates the need for a reboot.
func (k *Key) evalSubKeys() error {

	// error condition; the handle should already be in place by the time this
	// method is called.
	if k.runtime.handle == nil {
		return fmt.Errorf(
			"required handle to registry key %s is not open: %w",
			k,
			ErrKeyNotOpen,
		)
	}

	// Only check for subkeys if we are using their presence to indicate a
	// required reboot.
	switch {
	case k.ExpectedEvidence().SubKeysExist:

		logger.Printf("SubKeysExist specified; checking for subkeys for %q", k)

		// Fetch subkey names and record as matched paths.
		subKeyNames, err := k.runtime.handle.ReadSubKeyNames(0)
		if err != nil {
			return fmt.Errorf(
				"unexpected error occurred while retrieving subkey names for key %s: %w",
				k,
				err,
			)
		}

		logger.Printf("%d subkeys found for key %q", len(subKeyNames), k)

		if len(subKeyNames) > 0 {
			logger.Println("Reboot Evidence found!")
			k.SetFoundEvidenceSubKeysExist()

			relativePathSubKeyNames := make([]string, 0, len(subKeyNames))
			for _, subKeyName := range subKeyNames {
				relativePathSubKeyNames = append(
					relativePathSubKeyNames, filepath.Join(
						k.path, subKeyName,
					))
			}

			k.AddMatchedPath(relativePathSubKeyNames...)

			return nil
		}

	default:
		logger.Printf("SubKeysExist not specified; skipped checking for subkeys for %q", k)
	}

	return nil
}

// evalValue performs the tasks needed to evaluate whether the presence of a
// given registry key value indicates the need for a reboot.
func (k *Key) evalValue() error {

	// error condition; the handle should already be in place by the time this
	// method is called.
	if k.runtime.handle == nil {
		return fmt.Errorf(
			"required handle to registry key %s is not open: %w",
			k,
			ErrKeyNotOpen,
		)
	}

	if k.Value() == "" {
		logger.Printf("Value NOT specified for key %q", k)
		return nil
	}

	logger.Printf("Value %q specified for key %q", k.Value(), k)

	_, valTypeCode, err := k.runtime.handle.GetValue(k.Value(), nil)
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if k.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but marked as required.", k.Value())
			return fmt.Errorf(
				"value %s not found, but marked as required: %w",
				k.Value(),
				restart.ErrMissingValue,
			)
		}

		logger.Printf("Value %q not found, but not marked as required.", k.Value())
		return nil

	case err != nil:
		valReqLabel := KeyReqOptionalLabel
		if k.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			k,
			err,
		)

		return fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			k.Value(),
			err,
		)

	}

	valType := getValueType(valTypeCode)
	k.runtime.valueType = valType

	logger.Printf(
		"Value %q of type %q for key %q found!", k.Value(), valType, k)
	if k.ExpectedEvidence().ValueExists {
		logger.Println("Reboot Evidence found!")
		k.SetFoundEvidenceValueExists()

		logger.Printf("Recording matched path %s", k.Path())
		k.AddMatchedPath(k.Path())

		return nil
	}

	return nil

}

// Err returns the error (if any) associated with evaluating the Key. Whether
// the Key has been marked as ignored is not considered.
func (k *Key) Err() error {
	return k.runtime.err
}

// Ignored indicates whether the Key has been marked as ignored.
//
// For the entire key to be ignored, this means that *all* recorded matched
// path entries have to be marked as ignored.
func (k *Key) Ignored() bool {

	numMatchedPaths := len(k.runtime.pathsMatched)

	// logger.Printf("%d pathsMatched entries for %q", numMatchedPaths, k)

	// An empty collection of entries can occur if an error occurred or if no
	// assertions were matched.
	if numMatchedPaths == 0 {
		// logger.Printf("%d pathsMatched entries for %q", numMatchedPaths, k)
		return false
	}

	for _, v := range k.runtime.pathsMatched {
		if !v.ignored {
			// logger.Printf("%s is not marked as ignored\n", v)
			return false
		}

		logger.Printf("%s is marked as ignored\n", v)
	}

	// The entire Key is ignored *only* if all recorded matched path entries
	// are marked as ignored.
	return true
}

// HasIgnored indicates whether any matched path for the Key have been marked
// as ignored.
func (k *Key) HasIgnored() bool {
	for _, v := range k.runtime.pathsMatched {
		if v.ignored {
			return true
		}
	}

	return false
}

// HasSubPathMatches indicates whether the Key has evidence of subkey matches.
func (k *Key) HasSubPathMatches() bool {
	return k.runtime.evidenceFound.SubKeysExist
}

// RebootRequired indicates whether an evaluation determined that a reboot is
// needed. If the Key has been marked as ignored (all recorded matched paths
// marked as ignored) the need for a reboot is not indicated.
func (k *Key) RebootRequired() bool {
	if !k.Ignored() && k.HasEvidence() {
		return true
	}

	return false
}

// IsCriticalState indicates whether an evaluation determined that the Key is
// in a CRITICAL state. Whether the Key has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (k *Key) IsCriticalState() bool {
	switch {

	// If we could determine that a reboot is required we consider that to be
	// a WARNING state.
	case !k.Ignored() && k.RebootRequired():
		return false

	// If we were unable to determine whether a reboot is required due to
	// errors we consider that to be a CRITICAL state *unless* it is a very
	// specific sentinel error.
	case !k.Ignored() && k.Err() != nil:
		if errors.Is(k.Err(), restart.ErrMissingOptionalItem) {
			return false
		}
		return true

	// No reboot required and no errors, not CRITICAL state.
	default:
		return false

	}
}

// IsWarningState indicates whether an evaluation determined that the Key is
// in a WARNING state. Whether the Key has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (k *Key) IsWarningState() bool {
	return !k.Ignored() && k.RebootRequired()
}

// IsOKState indicates whether an evaluation determined that the Key is in an
// OK state. Whether the Key has been marked as Ignored is considered. The
// caller is responsible for filtering the collection prior to calling this
// method.
// TODO: Cleanup the logic.
func (k *Key) IsOKState() bool {
	switch {
	case k.Ignored():
		logger.Printf("%q has ignored flag set; returning true", k)
		return true
	case !k.Ignored() && k.RebootRequired():
		logger.Printf("%q does not have ignored flag set, has RebootRequired; returning false", k)
		return false

	// TODO: Pull this out and expose via helper method to determine if error
	// can be safely skipped. Perhaps don't return an error at all for missing
	// optional items?
	case !k.Ignored() && k.Err() != nil:
		if errors.Is(k.Err(), restart.ErrMissingOptionalItem) {
			logger.Printf("%q does not have ignored flag set, has ErrMissingOptionalItem error; returning true (OK state)", k)
			return true
		}
		logger.Printf("%q does not have ignored flag set, has error other than ErrMissingOptionalItem; returning false", k)
		logger.Printf("%q has error: %s", k, k.Err())
		return false

	default:
		logger.Printf("%q does not match other case statements; returning true (OK state)", k)
		return true
	}
}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (k *Key) Evaluate() {
	k.evaluate(true)
}

// Filter uses the list of specified ignore patterns to mark each matched path
// for the Key as ignored *IF* a match is found.
//
// While matched path and ignored pattern entries are normalized before
// comparison, we record path entries using the original non-normalized form.
//
// If no matched paths are recorded Filter makes no changes. Filter should be
// called before performing final state evaluation.
func (k *Key) Filter(ignorePatterns []string) {

	numIgnorePatterns := len(ignorePatterns)
	var numIgnorePatternsApplied int

	if numIgnorePatterns == 0 {
		logger.Printf("0 ignore patterns specified for %q; skipping Filter", k)
		return
	}

	logger.Printf(
		"%d ignore patterns specified for %q; applying Filter",
		numIgnorePatterns,
		k,
	)

	for originalPathString, matchedPath := range k.runtime.pathsMatched {
		logger.Printf("Searching matched path %q for ignore pattern matches", originalPathString)

		normalizedPathString := textutils.NormalizePath(originalPathString)
		logger.Printf("Normalizing matched path %q as %q", originalPathString, normalizedPathString)

		for _, ignorePattern := range ignorePatterns {

			normalizedIgnorePattern := textutils.NormalizePath(ignorePattern)
			logger.Printf("Normalizing ignore pattern %q as %q", ignorePattern, normalizedIgnorePattern)

			if strings.Contains(normalizedPathString, normalizedIgnorePattern) {
				logger.Printf("matchedPath %q contains ignorePattern %q", originalPathString, ignorePattern)
				logger.Printf("marking matched path %q as ignored", originalPathString)

				matchedPath.ignored = true
				k.runtime.pathsMatched[originalPathString] = matchedPath
				numIgnorePatternsApplied++
			}
		}
	}

	logger.Printf("%d ignore patterns applied for %q", numIgnorePatternsApplied, k)
}

// HasEvidence indicates whether any evidence was found for an assertion
// evaluation.
func (k Keys) HasEvidence() bool {
	for _, key := range k {
		if key.HasEvidence() {
			return true
		}
	}

	return false
}

// Validate performs basic validation of all items in the collection. An error
// is returned for any validation failures.
func (k Keys) Validate() error {
	for _, key := range k {
		if err := key.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate performs basic validation for all items in the collection. An
// error is returned for any validation failures.
func (krr KeysRebootRequired) Validate() error {
	for _, k := range krr {
		if err := k.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Data returns the actual data stored for a registry key value.
func (kb *KeyBinary) Data() []byte {
	return kb.runtime.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (kb *KeyBinary) ExpectedData() []byte {
	return kb.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (kb *KeyBinary) DataDisplay() string {
	// TODO: Apply specific formatting to match how Windows binary registry
	// values are usually displayed.
	return fmt.Sprintf("%v", kb.Data())
}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (kb *KeyBinary) Evaluate() {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	kb.evaluate(false)

	defer kb.closeAndLog()

	// Go no further if an error occurred evaluating the "base" Key.
	if kb.Err() != nil {
		return
	}
	// Go no further if there isn't a registry key value to process.
	if kb.Value() == "" {
		return
	}

	foundData, _, err := kb.Handle().GetBinaryValue(kb.Value())
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if kb.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but marked as required.", kb)
			kb.Key.runtime.err = fmt.Errorf(
				"value %s not found, but marked as required: %w",
				kb.Value(),
				restart.ErrMissingValue,
			)

			return
		}

		logger.Printf("Value %q not found, but not marked as required.", kb.Value())

		return

	case err != nil:

		valReqLabel := KeyReqOptionalLabel
		if kb.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			kb,
			err,
		)

		kb.Key.runtime.err = fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			kb.Value(),
			err,
		)

		return
	}

	logger.Printf("Data for value %q retrieved ...", kb.Value())
	logger.Printf("foundData: %v", foundData)
	logger.Print("Saving retrieved data for later use ...")
	kb.runtime.data = append(kb.runtime.data, foundData...)

	if !bytes.Equal(foundData, kb.ExpectedData()) {
		logger.Printf("%v does not match %v", foundData, kb.Data())

		// Only indicate that a reboot is required if the Key was marked
		// as we're considering a mismatch to be evidence. While unlikely,
		// we may wish to include Key values in our list that we are not
		// 100% certain indicate a need for a reboot.
		if kb.ExpectedEvidence().DataOtherThanX {
			logger.Println("Reboot Evidence found!")
			kb.SetFoundEvidenceDataOtherThanX()

			logger.Printf("Recording matched path %s", kb.Path())
			kb.AddMatchedPath(kb.Path())

			return
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary.
}

// Data returns the actual data stored for a registry key value.
func (ki *KeyInt) Data() uint64 {
	return ki.runtime.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (ki *KeyInt) ExpectedData() uint64 {
	return ki.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ki *KeyInt) DataDisplay() string {
	return fmt.Sprintf("%v", ki.Data())
}

// Evaluate performs evaluation of the embedded Key value and then applies
// (optional) evaluation of the data field.
func (ki *KeyInt) Evaluate() {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	ki.evaluate(false)

	defer ki.closeAndLog()

	// Go no further if an error occurred evaluating the "base" Key.
	if ki.Err() != nil {
		return
	}

	// Go no further if there isn't a registry key value to process.
	if ki.Value() == "" {
		return
	}

	foundData, _, err := ki.Handle().GetIntegerValue(ki.Value())
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if ki.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but marked as required.", ki)

			ki.Key.runtime.err = fmt.Errorf(
				"value %s not found, but marked as required: %w",
				ki.Value(),
				restart.ErrMissingValue,
			)

			return
		}

		logger.Printf("Value %q not found, but not marked as required.", ki.Value())

		return

	case err != nil:

		valReqLabel := KeyReqOptionalLabel
		if ki.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			ki,
			err,
		)

		ki.Key.runtime.err = fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			ki.Value(),
			err,
		)

		return
	}

	logger.Printf("Data for value %q retrieved ...", ki.Value())
	logger.Printf("foundData: %v", foundData)
	logger.Print("Saving retrieved data for later use ...")
	ki.runtime.data = foundData

	if foundData != ki.ExpectedData() {
		logger.Printf("%v does not match %v", foundData, ki.Data())

		// Only indicate that a reboot is required if the Key was marked
		// as we're considering a mismatch to be evidence. While unlikely,
		// we may wish to include Key values in our list that we are not
		// 100% certain indicate a need for a reboot.
		if ki.ExpectedEvidence().DataOtherThanX {
			logger.Println("Reboot Evidence found!")
			ki.SetFoundEvidenceDataOtherThanX()

			logger.Printf("Recording matched path %s", ki.Path())
			ki.AddMatchedPath(ki.Path())

			return
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary.
}

// Data returns the actual data stored for a registry key value.
func (ks *KeyString) Data() string {
	return ks.runtime.data
}

// ExpectedData returns the actual data stored for a registry key value.
func (ks *KeyString) ExpectedData() string {
	return ks.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ks *KeyString) DataDisplay() string {
	return fmt.Sprintf("%v", ks.Data())
}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (ks *KeyString) Evaluate() {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	ks.evaluate(false)

	defer ks.closeAndLog()

	// Go no further if an error occurred evaluating the "base" Key.
	if ks.Err() != nil {
		return
	}

	// Go no further if there isn't a registry key value to process.
	if ks.Value() == "" {
		return
	}

	foundData, _, err := ks.Handle().GetStringValue(ks.Value())
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if ks.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but is marked as required.", ks.Value())

			ks.Key.runtime.err = fmt.Errorf(
				"value %s not found, but is marked as required: %w",
				ks.Value(),
				restart.ErrMissingValue,
			)

			return
		}

		logger.Printf("Value %q not found, but not marked as required.", ks.Value())

		return

	case err != nil:

		valReqLabel := KeyReqOptionalLabel
		if ks.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			ks,
			err,
		)

		ks.Key.runtime.err = fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			ks.Value(),
			err,
		)

		return
	}

	logger.Printf("Data for value %q retrieved ...", ks.Value())
	logger.Printf("foundData: %v", foundData)
	logger.Print("Saving retrieved data for later use ...")
	ks.runtime.data = foundData

	if foundData != ks.ExpectedData() {
		logger.Printf("%v does not match %v", foundData, ks.ExpectedData())

		// Only indicate that a reboot is required if the Key was marked
		// as we're considering a mismatch to be evidence. While unlikely,
		// we may wish to include Key values in our list that we are not
		// 100% certain indicate a need for a reboot.
		if ks.ExpectedEvidence().DataOtherThanX {
			logger.Println("Reboot Evidence found!")
			ks.SetFoundEvidenceDataOtherThanX()

			logger.Printf("Recording matched path %s", ks.Path())
			ks.AddMatchedPath(ks.Path())

			return
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary.
}

// Data returns the actual data stored for a registry key value.
func (ks *KeyStrings) Data() []string {
	return ks.runtime.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (ks *KeyStrings) ExpectedData() []string {
	return ks.expectedData
}

// CleanedData returns a copy of the data stored for a registry key value with
// patterns found to be problematic for display/logging removed. The original
// values are not modified.
func (ks *KeyStrings) CleanedData() []string {

	// Clone original values
	// cleaned := make([]string, len(ks.runtime.data), cap(ks.runtime.data))
	// copy(cleaned, ks.runtime.data)

	// Opted to not clone the original collection just in case it contains a
	// lot of entries that require cleanup.
	cleaned := make([]string, 0, len(ks.runtime.data))

	for _, entry := range ks.runtime.data {
		// Skip blank lines
		if strings.TrimSpace(entry) == "" {
			continue
		}

		// Remove common prefix found in PendingFileRenameOperations registry
		// key values.
		entry = strings.ReplaceAll(
			entry, pendingFileRenameOperationsPrefix, ``)

		cleaned = append(cleaned, entry)
	}

	return cleaned
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ks *KeyStrings) DataDisplay() string {
	logger.Printf("Called for %+v", ks)

	entriesFound := len(ks.runtime.data)
	var entriesSkipped int

	logger.Printf(
		"%d data entries found for key %q",
		entriesFound,
		ks.path,
	)

	// Return a subset of the data collection instead of the full set; real
	// world testing found close to 200 entries for a
	// PendingFileRenameOperations collection.
	switch {
	case entriesFound > RegKeyTypeMultiSZDataDisplayLimit:

		entriesSkipped = entriesFound - RegKeyTypeMultiSZDataDisplayLimit

		logger.Printf(
			"DataDisplay limit of %d exceeded: %d entries skipped",
			RegKeyTypeMultiSZDataDisplayLimit,
			entriesSkipped,
		)

		listPrefix := fmt.Sprintf(
			"Entries [%d total, %d skipped]", entriesFound, entriesSkipped,
		)

		samples := ks.CleanedData()[:RegKeyTypeMultiSZDataDisplayLimit]

		// samples = append([]string{warning}, samples...)
		listing := fmt.Sprintf(
			"%s: %s", listPrefix, strings.Join(samples, ", "))

		return listing

	default:

		listPrefix := fmt.Sprintf(
			"Entries [%d total, %d skipped]", entriesFound, entriesSkipped,
		)

		listing := fmt.Sprintf(
			"%s: %s", listPrefix, strings.Join(ks.CleanedData(), ", "))

		return listing
	}

}

// AdditionalEvidence indicates what additional evidence "markers" have been
// supplied. If the reboot evidence markers for the Key type are not matched,
// these  (also optional) set of evidence markers are then checked to
// determine if a reboot is required.
func (ks *KeyStrings) AdditionalEvidence() KeyStringsRebootEvidence {
	return ks.additionalEvidence
}

// RebootReasons returns a list of the reasons associated with the evidence
// found for an evaluation that indicates a reboot is needed.
func (ks *KeyStrings) RebootReasons() []string {

	// Gather existing reasons for a reboot so that we can (potentially)
	// expand on them with additional reasons.
	reasons := ks.Key.RebootReasons()

	if ks.runtime.evidenceFound.ValueFound {
		switch {
		case ks.runtime.searchTermMatched != "":
			reasons = append(reasons, fmt.Sprintf(
				"Found match %s in data for value %s of key %s",
				ks.runtime.searchTermMatched,
				ks.Value(),
				ks,
			))
		default:
			logger.Print("BUG: searchTerm should have been recorded for a single value match")
			reasons = append(reasons, fmt.Sprintf(
				"Found match in data for value %s of key %s",
				ks.Value(),
				ks,
			))
		}
	}

	if ks.runtime.evidenceFound.AllValuesFound {
		reasons = append(reasons, fmt.Sprintf(
			"All specified strings found in data for value %s of key %s",
			ks.Value(),
			ks,
		))
	}

	return reasons
}

// SetFoundEvidenceValueFound records that the ValueFound reboot evidence was
// found.
func (ks *KeyStrings) SetFoundEvidenceValueFound() {
	logger.Printf("Recording that the ValueFound evidence was found for %q", ks)
	ks.runtime.evidenceFound.ValueFound = true
}

// SetFoundEvidenceAllValuesFound records that the AllValuesFound reboot
// evidence was found.
func (ks *KeyStrings) SetFoundEvidenceAllValuesFound() {
	logger.Printf("Recording that the AllValuesFound evidence was found for %q", ks)
	ks.runtime.evidenceFound.AllValuesFound = true
}

// HasEvidence indicates whether any evidence was found for an assertion
// evaluation.
func (ks *KeyStrings) HasEvidence() bool {

	// Check enclosed Key first.
	if ks.Key.HasEvidence() {
		return true
	}

	if ks.runtime.evidenceFound.ValueFound {
		return true
	}

	if ks.runtime.evidenceFound.AllValuesFound {
		return true
	}

	return false
}

// evalExpectedData evaluates the expected data stored for a registry key
// value against the actual data found during the Evaluate method call.
func (ks *KeyStrings) evalExpectedData() {

	var valuesFound int
	for _, searchTerm := range ks.ExpectedData() {
		switch {
		case textutils.InList(searchTerm, ks.runtime.data, true):
			valuesFound++

			ks.runtime.searchTermMatched = searchTerm

			logger.Printf("Found match %q within %v", searchTerm, ks.Data())

			// If we are just looking for one value, go ahead and return
			// early without checking for other matches.
			if ks.AdditionalEvidence().ValueFound {
				logger.Println("Reboot Evidence found!")
				ks.SetFoundEvidenceValueFound()

				logger.Printf("Recording matched path %s", ks.Path())

				// NOTE: If we were not deduping collected path values
				// this would likely cause a bug.
				ks.AddMatchedPath(ks.Path())

				return
			}

		default:
			logger.Printf("No matches found for %v", searchTerm)
		}
	}

	if ks.AdditionalEvidence().AllValuesFound {
		if valuesFound == len(ks.ExpectedData()) {
			// 100% match: All specified string values were found.
			logger.Println("Reboot Evidence found!")
			ks.SetFoundEvidenceAllValuesFound()

			logger.Printf("Recording matched path %s", ks.Path())
			ks.AddMatchedPath(ks.Path())

			return
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary.
}

// Evaluate performs evaluation of the embedded Key value and then applies
// (optional) evaluation of the data field to determine whether any of the
// specified strings are found in the retrieved key value data. Any single
// match indicates a reboot is needed.
func (ks *KeyStrings) Evaluate() {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	ks.evaluate(false)

	defer ks.closeAndLog()

	// Go no further if an error occurred evaluating the "base" Key.
	if ks.Err() != nil {
		return
	}

	if ks.Value() == "" {
		return
	}

	foundData, _, err := ks.Handle().GetStringsValue(ks.Value())
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if ks.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but marked as required.", ks.Value())

			ks.Key.runtime.err = fmt.Errorf(
				"value %s not found, but marked as required: %w",
				ks.Value(),
				restart.ErrMissingValue,
			)

			return
		}

		logger.Printf("Value %q not found, but not marked as required.", ks.Value())

		return

	case err != nil:

		valReqLabel := KeyReqOptionalLabel
		if ks.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			ks,
			err,
		)

		ks.Key.runtime.err = fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			ks.Value(),
			err,
		)

		return
	}

	logger.Printf("Data for value %q retrieved ...", ks.Value())
	logger.Printf("foundData: %v", foundData)
	logger.Printf("searchTerms: %v", ks.Data())
	logger.Print("Saving retrieved data for later use ...")
	ks.runtime.data = append(ks.runtime.data, foundData...)

	ks.evalExpectedData()

}

// Data returns the actual data stores for both registry key values.
func (kp *KeyPair) Data() [][]byte {
	return kp.runtime.data
}

// DataDisplay provides a string representation of the data for both registry
// key values.
func (kp *KeyPair) DataDisplay() string {

	// Don't panic, it is OK to not have stored data.
	if len(kp.Data()) != 2 {
		logger.Printf("Length of %d for kp.Data()", len(kp.Data()))
		return ""
	}

	key1Val := kp.Data()[0]
	key2Val := kp.Data()[1]

	// FIXME: This doesn't produce a "clean" string. Instead when printed there
	// are gaps between characters.
	key1ValAsString := string(key1Val)
	key2ValAsString := string(key2Val)

	return strings.Join(
		[]string{
			key1ValAsString,
			key2ValAsString,
		},
		", ",
	)
}

// AdditionalEvidence indicates what additional evidence "markers" have been
// supplied. If the reboot evidence markers for the Key type are not matched,
// these  (also optional) set of evidence markers are then checked to
// determine if a reboot is required.
func (kp *KeyPair) AdditionalEvidence() KeyPairRebootEvidence {
	return kp.additionalEvidence
}

// RebootReasons returns a list of the reasons associated with the evidence
// found for an evaluation that indicates a reboot is needed.
func (kp *KeyPair) RebootReasons() []string {

	// Gather existing reasons for a reboot so that we can (potentially)
	// expand on them with additional reasons.
	//
	// The usual scenario is one reason per evidence match per Key, so we opt
	// to initialize with two slots, presumably one reason per evaluated Key.
	reasons := make([]string, 0, len(kp.Keys))
	for _, key := range kp.Keys {
		reasons = append(reasons, key.RebootReasons()...)
	}

	if kp.runtime.evidenceFound.PairedValuesDoNotMatch {
		fqpath1 := fmt.Sprintf(`%s\%s`, kp.Keys[0].Path(), kp.Keys[0].Value())
		fqpath2 := fmt.Sprintf(`%s\%s`, kp.Keys[1].Path(), kp.Keys[1].Value())

		reasons = append(reasons, fmt.Sprintf(
			"Data mismatch for %s and %s",
			fqpath1,
			fqpath2,
		))
	}

	return reasons

}

// SetFoundEvidencePairedValuesDoNotMatch records that the
// PairedValuesDoNotMatch reboot evidence was found.
func (kp *KeyPair) SetFoundEvidencePairedValuesDoNotMatch() {
	logger.Printf("Recording that the PairedValuesDoNotMatch evidence was found for %q", kp)
	kp.runtime.evidenceFound.PairedValuesDoNotMatch = true
}

// AddMatchedPath records given paths as successful assertion matches for each
// enclosed Key value. Duplicate entries are ignored.
func (kp *KeyPair) AddMatchedPath(paths ...string) {
	for i := range kp.Keys {
		kp.Keys[i].AddMatchedPath(paths...)
	}
}

// MatchedPaths returns all recorded paths from successful assertion matches
// for each enclosed Key value.
func (kp *KeyPair) MatchedPaths() restart.MatchedPaths {
	matchedPaths := make(restart.MatchedPaths, 0,
		len(kp.Keys[0].runtime.pathsMatched)+
			len(kp.Keys[1].runtime.pathsMatched),
	)

	for _, key := range kp.Keys {
		matchedPaths = append(matchedPaths, key.MatchedPaths()...)
	}

	return matchedPaths
}

// Err exposes the first underlying error (if any) from enclosed Keys. Whether
// the Key has been marked as ignored is not considered.
//
// TODO: Should we handle this differently? Based on the current design we
// fail early; if processing the first Key fails there won't be an error
// recorded for the second Key. This *should* mean that the approach of
// returning the first error is stable ...
func (kp *KeyPair) Err() error {
	if kp.Keys[0].Err() != nil {
		return kp.Keys[0].Err()
	}

	if kp.Keys[1].Err() != nil {
		return kp.Keys[1].Err()
	}

	return nil
}

// HasEvidence indicates whether any evidence was found for an assertion
// evaluation.
func (kp *KeyPair) HasEvidence() bool {
	// Check enclosed Keys first.
	for _, key := range kp.Keys {
		if key.HasEvidence() {
			return true
		}
	}

	//	if kp.runtime.evidenceFound.PairedValuesDoNotMatch {
	//		return true
	//	}
	//
	// return false

	return kp.runtime.evidenceFound.PairedValuesDoNotMatch
}

// gatherKeyPairData retrieves the data for a registry Key out from a pair for
// evaluation.
func (kp *KeyPair) gatherKeyPairData(key *Key) {
	bufSize, valType, err := key.Handle().GetValue(key.Value(), nil)
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if key.Requirements().ValueRequired {
			logger.Printf("Value %q not found, but marked as required.", key.Value())

			key.runtime.err = fmt.Errorf(
				"value %s not found, but marked as required: %w",
				key.Value(),
				restart.ErrMissingValue,
			)

			return
		}

		logger.Printf("Value %q not found, but not marked as required.", key.Value())

		return

	case err != nil:

		valReqLabel := KeyReqOptionalLabel
		if key.Requirements().ValueRequired {
			valReqLabel = KeyReqRequiredLabel
		}

		logger.Printf(
			"Unexpected error occurred while retrieving %s value %q: %s",
			valReqLabel,
			key,
			err,
		)

		key.runtime.err = fmt.Errorf(
			"unexpected error occurred while retrieving %s value %s: %w",
			valReqLabel,
			key.Value(),
			err,
		)

		return
	}

	logger.Printf(
		"Required buffer size %d for value %v of type %v ...",
		bufSize,
		key.Value(),
		getValueType(valType),
	)

	buffer := make([]byte, bufSize)
	_, _, err = key.Handle().GetValue(key.Value(), buffer)

	// We intentionally use simpler error handling here since we just
	// evaluated whether the registry key value is required to be
	// present and have successfully retrieved the necessary buffer
	// size for the data associated with registry key value; if an
	// error occurs at this point it doesn't really matter what
	// requirement is being applied.
	//
	// TODO: Worth the complexity to implement a retry?
	if err != nil {
		key.runtime.err = fmt.Errorf(
			"failed to retrieve data for value %s: %w",
			key.Value(),
			err,
		)

		return
	}

	logger.Printf("data in raw/hex format: % x", buffer)
	logger.Printf("data in string format: %s", buffer)

	logger.Print("Saving retrieved data for later use ...")
	kp.runtime.data = append(kp.runtime.data, buffer)
}

// evalKeyPairData evaluates retrieved data values.
func (kp *KeyPair) evalKeyPairData() {

	fqpath1 := fmt.Sprintf(`%s\%s`, kp.Keys[0].Path(), kp.Keys[0].Value())
	fqpath2 := fmt.Sprintf(`%s\%s`, kp.Keys[1].Path(), kp.Keys[1].Value())

	if !bytes.Equal(kp.runtime.data[0], kp.runtime.data[1]) {
		logger.Printf("Data for %q does not equal %q", fqpath1, fqpath2)
		logger.Println("Reboot Evidence found!")
		kp.SetFoundEvidencePairedValuesDoNotMatch()

		logger.Printf("Recording matched path %s", kp.Keys[0].Path())
		logger.Printf("Recording matched path %s", kp.Keys[1].Path())
		kp.AddMatchedPath(kp.Keys[0].Path(), kp.Keys[1].Path())

		return
	}

	logger.Printf("Data equal for %q and %q", fqpath1, fqpath2)
}

// Evaluate performs an evaluation of the key pair to determine whether a
// reboot is needed.
//
// TODO: Determine whether other KeyRebootEvidence values will be evaluated or
// just the PairedValuesDoNotMatch field per the ValuesMustMatch() method.
//
// TODO: At the time of initial development the intent is to support KeyPair
// representing both pairs of optional keys and those which are required to be
// present. Key values that are required to be present for each key in the
// pair is the primary focus as supporting that scenario is necessary for the
// initial implementation.
func (kp *KeyPair) Evaluate() {

	for _, key := range kp.Keys {

		// Evaluate embedded "base" Key first where we check shared
		// requirements and reboot evidence. We also explicitly indicate that
		// we wish to retain a handle to the open registry key (for use here).
		key.evaluate(false)

		defer key.closeAndLog()

		// Early exit logic kill switch.
		switch {
		case key.Err() != nil:
			// Go no further if an error occurred evaluating the "base" Key.
			//
			// Unlike other Key* types, this type is not meant to collect the
			// actual data registry values for later display purposes, only
			// make comparisons if requested. So, unlike the other Key* types,
			// it is sufficient to use only the base Key evaluation results if
			// an error occurred or a reboot was found to be required.
			return
		case key.HasEvidence():
			// Go no further if the "base" Key evaluation was sufficient to
			// determine a reboot is needed.
			return
		case !kp.AdditionalEvidence().PairedValuesDoNotMatch:
			// Exit early if we are not evaluating whether the data for each
			// registry key value matches.
			return
		case key.Value() == "":
			// Go no further if there isn't a registry key value to process.
			return
		}

		kp.gatherKeyPairData(key)

	}

	// compare retrieved data values
	kp.evalKeyPairData()
}

// Filter uses the list of specified ignore patterns to mark each matched path
// for the enclosed Keys as ignored *IF* a match is found. If no matched paths
// are recorded Filter makes no changes. Filter should be called before
// performing final state evaluation.
func (kp *KeyPair) Filter(ignorePatterns []string) {

	for i := range kp.Keys {
		kp.Keys[i].Filter(ignorePatterns)
	}
}

// Validate performs basic validation. An error is returned for any validation
// failures.
func (kp *KeyPair) Validate() error {

	if len(kp.Keys) != 2 {
		return fmt.Errorf(
			"%d paths specified: %w",
			len(kp.Keys),
			ErrInvalidNumberOfKeysInKeyPair,
		)
	}

	for _, key := range kp.Keys {

		// The general validation checks assert that at least one reboot
		// evidence field was specified.
		if err := key.Validate(); err != nil {
			return err
		}

	}

	return nil
}

func (kp *KeyPair) String() string {

	keys := make([]string, 0, len(kp.Keys))

	for _, key := range kp.Keys {
		keys = append(
			keys,
			fmt.Sprintf(
				`%v\%s`,
				getRootKeyName(key.RootKey()),
				key.Path(),
			))
	}

	return strings.Join(keys, ", ")
}

// Ignored indicates whether both Keys in the set have been marked as ignored.
func (kp *KeyPair) Ignored() bool {
	if kp.Keys[0].Ignored() && kp.Keys[1].Ignored() {
		return true
	}

	return false
}

// RebootRequired indicates whether an evaluation of both Keys in the set
// determined that a reboot is needed. If both Keys have been marked as
// ignored (all recorded matched paths marked as ignored) the need for a
// reboot is not indicated.
func (kp *KeyPair) RebootRequired() bool {
	if !kp.Ignored() && kp.HasEvidence() {
		return true
	}

	return false
}

// IsCriticalState indicates whether an evaluation determined that the KeyPair
// is in a CRITICAL state. Whether the KeyPair has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (kp *KeyPair) IsCriticalState() bool {
	for _, key := range kp.Keys {
		if key.IsCriticalState() {
			return true
		}
	}

	return false
}

// IsWarningState indicates whether an evaluation determined that the KeyPair
// is in a WARNING state. Whether the KeyPair has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (kp *KeyPair) IsWarningState() bool {
	return !kp.Ignored() && kp.RebootRequired()
}

// IsOKState indicates whether an evaluation determined that the KeyPair is in
// an OK state. Whether the KeyPair has been marked as Ignored is considered.
// The caller is responsible for filtering the collection prior to calling
// this method. TODO: Cleanup the logic.
func (kp *KeyPair) IsOKState() bool {
	switch {
	case kp.Ignored():
		return true
	case !kp.Ignored() && kp.RebootRequired():
		return false
	case !kp.Ignored() && kp.Err() != nil:
		return false
	default:
		return true
	}
}

// getType returns a string description for the given registry key value type.
//
// TODO: Export for external use?
func getValueType(valType uint32) string {
	var keyType string

	switch valType {
	case registry.NONE:
		keyType = RegKeyTypeNone
	case registry.SZ:
		keyType = RegKeyTypeSZ
	case registry.EXPAND_SZ:
		keyType = RegKeyTypeExpandSZ
	case registry.BINARY:
		keyType = RegKeyTypeBinary
	case registry.DWORD:
		keyType = RegKeyTypeDWORD
	case registry.DWORD_BIG_ENDIAN:
		keyType = RegKeyTypeDWORDBigEndian
	case registry.LINK:
		keyType = RegKeyTypeLink
	case registry.MULTI_SZ:
		keyType = RegKeyTypeMultiSZ
	case registry.RESOURCE_LIST:
		keyType = RegKeyTypeResourceList
	case registry.FULL_RESOURCE_DESCRIPTOR:
		keyType = RegKeyTypeFullResourceDescriptor
	case registry.RESOURCE_REQUIREMENTS_LIST:
		keyType = RegKeyTypeResourceRequirementsList
	case registry.QWORD:
		keyType = RegKeyTypeQWORD
	default:
		keyType = RegKeyTypeUnknown
	}

	return keyType
}

// getRootKeyName returns a string description for the given registry root
// key.
//
// TODO: Export for external use?
func getRootKeyName(key registry.Key) string {
	var keyName string

	switch key {
	case registry.CLASSES_ROOT:
		keyName = RegKeyRootNameClassesRoot
	case registry.CURRENT_USER:
		keyName = RegKeyRootNameCurrentUser
	case registry.LOCAL_MACHINE:
		keyName = RegKeyRootNameLocalMachine
	case registry.USERS:
		keyName = RegKeyRootNameUsers
	case registry.CURRENT_CONFIG:
		keyName = RegKeyRootNameCurrentConfig
	case registry.PERFORMANCE_DATA:
		keyName = RegKeyRootNamePerformanceData
	default:
		keyName = RegKeyRootNameUnknown
	}

	return keyName
}

// func matchedPathsFromPathStrings(rootPath string, pathStrings []string) restart.MatchedPaths {
// 	matchedPaths := make(restart.MatchedPaths, 0, len(pathStrings))
//
// 	for _, path := range sort.StringSlice(pathStrings) {
//
// 		matchedPath := MatchedPath{
// 			root:     rootPath,
// 			relative: path,
// 			base:     filepath.Base(path),
// 		}
//
// 		matchedPaths = append(matchedPaths, matchedPath)
// 	}
//
// 	return matchedPaths
//
// }

// func matchedPathFromPathString(rootPath string, pathString string) restart.MatchedPath {
//
// 	relPath, err := filepath.Rel(rootPath, pathString)
// 	switch {
// 	case err != nil:
// 		logger.Printf("Failed to obtain relative path for %q using %q as the base", pathString, rootPath)
// 		logger.Printf("Falling back to using %q as relative path", pathString)
// 		relPath = pathString
// 	default:
// 		logger.Printf(
// 			"Successfully resolved %q as relative path of %q using %q as root path",
// 			relPath,
// 			pathString,
// 			rootPath,
// 		)
// 	}
//
// 	matchedPath := MatchedPath{
// 		root:     rootPath,
// 		relative: relPath,
// 		base:     filepath.Base(pathString),
// 	}
//
// 	return matchedPath
// }
