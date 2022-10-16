//go:build windows
// +build windows

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

// KeyRebootEvidence indicates what registry key evidence is required in order
// to determine that a reboot is needed.
type KeyRebootEvidence struct {
	DataOtherThanX bool
	SubKeysExist   bool
	ValueExists    bool
	KeyExists      bool
}

// KeyPairRebootEvidence applies additional evidence "markers" for the KeyPair
// type. If the reboot evidence markers for the enclosed Keys are not matched,
// this (also optional) evidence marker is then checked to determine if a
// reboot is required for the pair as a whole.
type KeyPairRebootEvidence struct {
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

// Key represents a registry key that if found (and requirements met)
// indicates a reboot is needed.
type Key struct {
	// root is the root or base registry key (e.g, HKEY_LOCAL_MACHINE).
	root registry.Key

	// handle is a handle to an open registry key. This is set by the Open
	// method on this type and required to be present by the Evaluate methods
	// on this type and the super types embedding this type.
	//
	// Per official documentation, a handle to an open registry key should not
	// be used after it is closed and should not remain open any longer than
	// necessary.
	// https://learn.microsoft.com/en-us/windows/win32/api/winreg/nf-winreg-regclosekey
	handle *registry.Key

	// skipClosingHandle indicates whether an open registry key handle should
	// be left open at the end of an operation. Unless directed otherwise the
	// default behavior for methods that open a registry key will be to
	// cleanup by closing the handle at the end of the method call.
	//
	// Setting this field allows the handle to remain valid for later use.
	// This is often used by super types that embed this type which need to
	// perform additional operations using the handle.
	// skipClosingHandle bool

	// path is the registry key path minus the root key (e.g.,
	// HKEY_LOCAL_MACHINE) and any value to be evaluated.
	path string

	// value is the registry key value name.
	value string

	// evidence indicates what is required in order to determine that a reboot
	// is needed.
	evidence KeyRebootEvidence

	// requirements indicates what requirements must be met. If not met, this
	// indicates that an error has occurred.
	requirements KeyAssertions
}

// Keys is a collection of Key values.
type Keys []Key

// KeysRebootRequired is a collection of KeysRebootRequired values.
type KeysRebootRequired []KeyRebootRequired

// KeyPair represents two Keys that are evaluated together.
type KeyPair struct {
	Keys               Keys
	additionalEvidence KeyPairRebootEvidence

	// data represents the data stored for both registry key values.
	data [][]byte
}

// KeyInt represents a Key containing integer data for comparison.
type KeyInt struct {
	Key

	// data represents the data stored for a registry key value.
	data uint64

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData uint64
}

// KeyBinary represents a Key containing binary data for comparison.
type KeyBinary struct {
	Key

	// data represents the actual data stored for a registry key value.
	data []byte

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData []byte
}

// KeyString represents a Key containing string data for comparison.
type KeyString struct {
	Key

	// data represents the actual data stored for a registry key value.
	data string

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData string
}

// KeyStrings represents a Key containing multiple strings for comparison.
// That collection of strings maps to a registry.MULTI_SZ key type.
//
// Multiple strings are stored in the registry Data file as null terminated
// strings, but are retrieved as a slice of strings.
type KeyStrings struct {
	Key

	// data represents the actual data stored for a registry key value.
	data []string

	// expectedData represents the data that will be compared against the
	// actual data stored for a registry key value.
	expectedData []string

	// additionalEvidence applies additional evidence "markers" for this type.
	// KeyStrings type. If the reboot evidence markers for the enclosed Key
	// type are not matched, this (also optional) set of evidence markers are
	// then checked to determine if a reboot is required.
	additionalEvidence KeyStringsRebootEvidence
}

// Evidence returns the specified evidence that is required in order to
// determine that a reboot is needed.
func (k Key) Evidence() KeyRebootEvidence {
	return k.evidence
}

func (k Key) String() string {

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
func (k Key) Requirements() KeyAssertions {
	return k.requirements
}

// Path returns the specified registry key path.
func (k Key) Path() string {
	return k.path
}

// RootKey returns the specified registry root key.
func (k Key) RootKey() registry.Key {
	return k.root
}

// Value returns the specified registry key value.
func (k Key) Value() string {
	return k.value
}

// Handle returns the current handle to the open registry key if it exists,
// otherwise returns nil.
//
// TODO: Should this be provided?
func (k Key) Handle() *registry.Key {
	return k.handle
}

// open creates a handle to the registry key and saves it for later use. The
// caller is responsible for calling the Close method to free the resources
// used by the open registry key.
func (k *Key) open() error {
	// Skip opening a handle to the registry key if it is already open.
	if k.handle != nil {
		logger.Printf("Handle exists; registry key %q is already open", k)
		return ErrKeyAlreadyOpen
	}

	logger.Printf("Handle does not exist, attempting to open registry key %q", k)

	// NOTE: If we wish to enumerate subkeys we should request access
	// to do so along with permission to query values.
	//
	// We specify both permissions by combining the values via OR.
	// https://stackoverflow.com/questions/47814070/golang-cant-enumerate-subkeys-of-registry-key
	// k, err := registry.OpenKey(key.RootKey(), path, registry.QUERY_VALUE|registry.ENUMERATE_SUB_KEYS)

	// We do not request access to enumerate subkeys because we can get the
	// needed subkey count by calling Stat on the open key.
	openKey, err := registry.OpenKey(k.RootKey(), k.Path(), registry.QUERY_VALUE)
	switch {
	case errors.Is(err, registry.ErrNotExist):
		if k.Requirements().KeyRequired {
			logger.Printf("Key %q not found, but marked as required.", k)
			return ErrMissingRequiredKey
		}

		logger.Printf("Key %q not found, but not marked as required.", k)
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

	k.handle = &openKey

	// TODO: Any other feasible way to handle this? This is a logic problem
	// that needs to be resolved.
	if k.handle == nil {
		panic("BUG: k.handle is nil and should not be. Explosions commence!")
	}

	return nil

}

// close will close the handle to a registry key if open, otherwise will
// act as a NOOP. An error is returned if one is encountered when attempting
// to close the handle.
func (k *Key) close() error {

	if k.handle == nil {
		logger.Printf("Handle for %s already closed", k)
		return nil
	}

	if err := k.handle.Close(); err != nil {
		logger.Printf("Error encountered closing handle to %s: %v", k, err)
		return err
	}

	// Remove reference to the handle since it is no longer valid.
	logger.Printf("Removed reference to the handle since it is no longer valid")
	k.handle = nil

	return nil

}

// Validate performs basic validation. An error is returned for any validation
// failures.
func (k Key) Validate() error {

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
	if k.Value() == "" && k.evidence.ValueExists {
		// logger.Printf("evidence: %+v", k.evidence)
		return fmt.Errorf(
			"required registry value not specified: %w",
			restart.ErrMissingValue,
		)
	}

	// Validate reboot evidence values.
	switch {
	case k.evidence.DataOtherThanX:
	case k.evidence.SubKeysExist:
	case k.evidence.ValueExists:
	case k.evidence.KeyExists:
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
func (k *Key) evaluate(closeHandle bool) restart.RebootCheckResult {
	logger.Printf("Evaluating key %q", k)

	evalOpenKeyResult := k.evalOpenKey()
	if evalOpenKeyResult.Err != nil {

		logger.Print("Evaluation of specified registry key unsuccessful")

		// Replace with general error value that the client code can more
		// easily use to determine severity.
		switch {
		case errors.Is(evalOpenKeyResult.Err, ErrMissingOptionalKey):
			evalOpenKeyResult.Err = restart.ErrMissingOptionalItem

		case errors.Is(evalOpenKeyResult.Err, ErrMissingRequiredKey):
			evalOpenKeyResult.Err = restart.ErrMissingRequiredItem
		}

		return evalOpenKeyResult
	}

	// Only attempt to close the handle if we successfully opened it.
	defer func() {
		if !closeHandle {
			logger.Printf("Skipping closure of handle to %q as requested", k)
			return
		}

		logger.Printf("Closing handle to %q", k)
		if err := k.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", k)
		}
	}()

	// If a reboot is needed skip any further checks.
	if evalOpenKeyResult.RebootRequired {
		return evalOpenKeyResult
	}

	evalValueResult := k.evalValue()
	if evalValueResult.Err != nil || evalValueResult.RebootRequired {
		return evalValueResult
	}

	evalSubKeysResult := k.evalSubKeys()
	if evalSubKeysResult.Err != nil || evalSubKeysResult.RebootRequired {
		return evalSubKeysResult
	}

	return restart.RebootCheckResult{
		Examined:       k,
		RebootRequired: false,
	}
}

// evalOpenKey performs the tasks needed to open a handle to the registry key
// and evaluate whether there is a need for a reboot. The handle to the open
// registry key is retained for later use.
//
// The caller is responsible for calling the close method to free resources
// used by the open registry key.
func (k *Key) evalOpenKey() restart.RebootCheckResult {

	logger.Printf("Opening key %q", k)

	err := k.open()
	switch {
	case errors.Is(err, ErrKeyAlreadyOpen):
		logger.Printf("Key %q is already open?", k)
		logger.Print("TODO: Probably worth checking how this occurred.")

		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"evalOpenKey() for key %s failed: %w", k, ErrKeyAlreadyOpen,
			),
		}

	case errors.Is(err, ErrMissingRequiredKey):
		logger.Printf("Key %q not found, but marked as required.", k)
		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"evalOpenKey() for key %s failed: %w", k, ErrMissingRequiredKey,
			),
		}

	case errors.Is(err, ErrMissingOptionalKey):
		logger.Printf("Key %q not found, but not marked as required.", k)
		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"evalOpenKey() for key %s unsuccessful: %w", k, ErrMissingOptionalKey,
			),
		}

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

		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"evalOpenKey() for key %s failed: %s: %w",
				keyReqLabel,
				k,
				err,
			),
		}

	default:

		logger.Printf("Key %q opened ...", k)

		if k.Evidence().KeyExists {
			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: true,
				RebootReasons: []string{
					fmt.Sprintf(
						"Key %s found", k,
					),
				},
			}
		}

		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
		}
	}

}

// evalSubKeys performs the tasks needed to evaluate whether the presence of
// subkeys for a given registry key indicates the need for a reboot.
func (k *Key) evalSubKeys() restart.RebootCheckResult {

	// error condition; the handle should already be in place by the time this
	// method is called.
	if k.handle == nil {
		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"required handle to registry key %s is not open: %w",
				k,
				ErrKeyNotOpen,
			),
		}
	}

	// Only check for subkeys if we are using their presence to indicate a
	// required reboot.
	switch {
	case k.Evidence().SubKeysExist:

		logger.Printf("SubKeysExist specified; checking for subkeys for %q", k)

		keyInfo, err := k.handle.Stat()
		if err != nil {
			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving info for key %s: %w",
					k,
					err,
				),
			}
		}

		logger.Printf("%d subkeys found for key %q", keyInfo.SubKeyCount, k)

		// There are keys, so a reboot is required.
		if keyInfo.SubKeyCount > 0 {
			logger.Println("Reboot Required!")
			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: true,
				RebootReasons: []string{
					fmt.Sprintf(
						"Subkeys for key %s found", k,
					),
				},
			}
		}

	default:
		logger.Printf("SubKeysExist not specified; skipped checking for subkeys for %q", k)
	}

	return restart.RebootCheckResult{
		Examined:       k,
		RebootRequired: false,
	}
}

// evalValue performs the tasks needed to evaluate whether the presence of a
// given registry key value indicates the need for a reboot.
func (k *Key) evalValue() restart.RebootCheckResult {

	// error condition; the handle should already be in place by the time this
	// method is called.
	if k.handle == nil {
		return restart.RebootCheckResult{
			Examined:       k,
			RebootRequired: false,
			Err: fmt.Errorf(
				"required handle to registry key %s is not open: %w",
				k,
				ErrKeyNotOpen,
			),
		}
	}

	if k.Value() != "" {

		logger.Printf("Value %q specified for key %q", k.Value(), k)

		_, valTypeCode, err := k.handle.GetValue(k.Value(), nil)
		switch {
		case errors.Is(err, registry.ErrNotExist):
			if k.Requirements().ValueRequired {
				logger.Printf("Value %q not found, but marked as required.", k.Value())
				return restart.RebootCheckResult{
					Examined:       k,
					RebootRequired: false,
					Err: fmt.Errorf(
						"value %s not found, but marked as required: %w",
						k.Value(),
						restart.ErrMissingValue,
					),
				}
			}

			logger.Printf("Value %q not found, but not marked as required.", k.Value())
			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: false,
			}

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

			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving %s value %s: %w",
					valReqLabel,
					k.Value(),
					err,
				),
			}
		}

		valType := getValueType(valTypeCode)

		logger.Printf(
			"Value %q of type %q for key %q found!", k.Value(), valType, k)
		if k.Evidence().ValueExists {
			logger.Println("Reboot Required!")
			return restart.RebootCheckResult{
				Examined:       k,
				RebootRequired: true,
				RebootReasons: []string{
					fmt.Sprintf(
						"Value %s of type %s for key %s found",
						k.Value(),
						valType,
						k,
					),
				},
			}
		}
	}

	logger.Printf("Value NOT specified for key %q", k)

	return restart.RebootCheckResult{
		Examined:       k,
		RebootRequired: false,
	}

}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (k Key) Evaluate() restart.RebootCheckResult {
	return k.evaluate(true)
}

// Validate performs basic validation of all items in the collection. An error
// is returned for any validation failures.
func (k Keys) Validate() error {
	for _, k := range k {
		if err := k.Validate(); err != nil {
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
func (kb KeyBinary) Data() []byte {
	return kb.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (kb KeyBinary) ExpectedData() []byte {
	return kb.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (kb KeyBinary) DataDisplay() string {
	// TODO: Apply specific formatting to match how Windows binary registry
	// values are usually displayed.
	return fmt.Sprintf("%v", kb.Data())
}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (kb KeyBinary) Evaluate() restart.RebootCheckResult {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	baseKeyEvalResult := kb.evaluate(false)

	defer func() {
		logger.Printf("Closing open handle to %q", kb)
		if err := kb.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", kb)
		}

		if kb.handle != nil {
			logger.Printf("Failed to close handle to %q", kb)
		}
		logger.Printf("Closed handle to %q", kb)
	}()

	// Go no further if an error occurred evaluating the "base" Key.
	if baseKeyEvalResult.Err != nil {
		return baseKeyEvalResult
	}

	if kb.Value() != "" {
		foundData, _, err := kb.handle.GetBinaryValue(kb.Value())
		switch {
		case errors.Is(err, registry.ErrNotExist):
			if kb.Requirements().ValueRequired {
				logger.Printf("Value %q not found, but marked as required.", kb)
				return restart.RebootCheckResult{
					Examined:       kb,
					RebootRequired: false,
					Err: fmt.Errorf(
						"value %s not found, but marked as required: %w",
						kb.Value(),
						restart.ErrMissingValue,
					),
				}
			}

			logger.Printf("Value %q not found, but not marked as required.", kb.Value())
			return restart.RebootCheckResult{
				Examined:       kb,
				RebootRequired: false,
			}

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

			return restart.RebootCheckResult{
				Examined:       kb,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving %s value %s: %w",
					valReqLabel,
					kb.Value(),
					err,
				),
			}
		}

		logger.Printf("Data for value %q retrieved ...", kb.Value())
		logger.Printf("foundData: %v", foundData)
		logger.Print("Saving retrieved data for later use ...")
		kb.data = append(kb.data, foundData...)

		if !bytes.Equal(foundData, kb.ExpectedData()) {
			logger.Printf("%v does not match %v", foundData, kb.Data())

			// Only indicate that a reboot is required if the Key was marked
			// as we're considering a mismatch to be evidence. While unlikely,
			// we may wish to include Key values in our list that we are not
			// 100% certain indicate a need for a reboot.
			if kb.Evidence().DataOtherThanX {
				logger.Println("Reboot Required!")
				return restart.RebootCheckResult{
					Examined:       kb,
					RebootRequired: true,
					RebootReasons: []string{
						fmt.Sprintf(
							"Data for value %s for key %s matches", kb.Value(), kb,
						),
					},
				}
			}
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary. If the earlier base Key
	// evaluation indicated that a reboot was needed, let's use that but with
	// a minor change to emphasize that this super type was examined.
	if baseKeyEvalResult.RebootRequired {
		result := baseKeyEvalResult
		result.Examined = kb

		return result
	}

	// Otherwise, fallback to a standard "no reboot required" result.
	return restart.RebootCheckResult{
		Examined:       kb,
		RebootRequired: false,
	}

}

// Data returns the actual data stored for a registry key value.
func (ki KeyInt) Data() uint64 {
	return ki.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (ki KeyInt) ExpectedData() uint64 {
	return ki.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ki KeyInt) DataDisplay() string {
	return fmt.Sprintf("%v", ki.Data())
}

// Evaluate performs evaluation of the embedded Key value and then applies
// (optional) evaluation of the data field.
func (ki KeyInt) Evaluate() restart.RebootCheckResult {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	baseKeyEvalResult := ki.evaluate(false)

	defer func() {
		logger.Printf("Closing open handle to %q", ki)
		if err := ki.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", ki)
		}

		if ki.handle != nil {
			logger.Printf("Failed to close handle to %q", ki)
		}
		logger.Printf("Closed handle to %q", ki)
	}()

	// Go no further if an error occurred evaluating the "base" Key.
	if baseKeyEvalResult.Err != nil {
		return baseKeyEvalResult
	}

	if ki.Value() != "" {

		foundData, _, err := ki.handle.GetIntegerValue(ki.Value())
		switch {
		case errors.Is(err, registry.ErrNotExist):
			if ki.Requirements().ValueRequired {
				logger.Printf("Value %q not found, but marked as required.", ki)
				return restart.RebootCheckResult{
					Examined:       ki,
					RebootRequired: false,
					Err: fmt.Errorf(
						"value %s not found, but marked as required: %w",
						ki.Value(),
						restart.ErrMissingValue,
					),
				}
			}

			logger.Printf("Value %q not found, but not marked as required.", ki.Value())
			return restart.RebootCheckResult{
				Examined:       ki,
				RebootRequired: false,
			}

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

			return restart.RebootCheckResult{
				Examined:       ki,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving %s value %s: %w",
					valReqLabel,
					ki.Value(),
					err,
				),
			}
		}

		logger.Printf("Data for value %q retrieved ...", ki.Value())
		logger.Printf("foundData: %v", foundData)
		logger.Print("Saving retrieved data for later use ...")
		ki.data = foundData

		if foundData != ki.ExpectedData() {
			logger.Printf("%v does not match %v", foundData, ki.Data())

			// Only indicate that a reboot is required if the Key was marked
			// as we're considering a mismatch to be evidence. While unlikely,
			// we may wish to include Key values in our list that we are not
			// 100% certain indicate a need for a reboot.
			if ki.Evidence().DataOtherThanX {
				logger.Println("Reboot Required!")
				return restart.RebootCheckResult{
					Examined:       ki,
					RebootRequired: true,
					RebootReasons: []string{
						fmt.Sprintf(
							"Data for value %s for key %s found", ki.Value(), ki,
						),
					},
				}
			}
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary. If the earlier base Key
	// evaluation indicated that a reboot was needed, let's use that but with
	// a minor change to emphasize that this super type was examined.
	if baseKeyEvalResult.RebootRequired {
		result := baseKeyEvalResult
		result.Examined = ki

		return result
	}

	// Otherwise, fallback to a standard "no reboot required" result.
	return restart.RebootCheckResult{
		Examined:       ki,
		RebootRequired: false,
	}

}

// Data returns the actual data stored for a registry key value.
func (ks KeyString) Data() string {
	return ks.data
}

// ExpectedData returns the actual data stored for a registry key value.
func (ks KeyString) ExpectedData() string {
	return ks.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ks KeyString) DataDisplay() string {
	return fmt.Sprintf("%v", ks.Data())
}

// Evaluate performs the minimum number of assertions to determine whether a
// reboot is needed. If an error is encountered further checks are skipped.
func (ks KeyString) Evaluate() restart.RebootCheckResult {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	baseKeyEvalResult := ks.evaluate(false)

	defer func() {
		logger.Printf("Closing open handle to %q", ks)
		if err := ks.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", ks)
		}

		if ks.handle != nil {
			logger.Printf("Failed to close handle to %q", ks)
		}
		logger.Printf("Closed handle to %q", ks)
	}()

	// Go no further if an error occurred evaluating the "base" Key.
	if baseKeyEvalResult.Err != nil {
		return baseKeyEvalResult
	}

	if ks.Value() != "" {
		foundData, _, err := ks.handle.GetStringValue(ks.Value())
		switch {
		case errors.Is(err, registry.ErrNotExist):
			if ks.Requirements().ValueRequired {
				logger.Printf("Value %q not found, but is marked as required.", ks.Value())
				return restart.RebootCheckResult{
					Examined:       ks,
					RebootRequired: false,
					Err: fmt.Errorf(
						"value %s not found, but is marked as required: %w",
						ks.Value(),
						restart.ErrMissingValue,
					),
				}
			}

			logger.Printf("Value %q not found, but not marked as required.", ks.Value())
			return restart.RebootCheckResult{
				Examined:       ks,
				RebootRequired: false,
			}

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

			return restart.RebootCheckResult{
				Examined:       ks,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving %s value %s: %w",
					valReqLabel,
					ks.Value(),
					err,
				),
			}
		}

		logger.Printf("Data for value %q retrieved ...", ks.Value())
		logger.Printf("foundData: %v", foundData)
		logger.Print("Saving retrieved data for later use ...")
		ks.data = foundData

		if foundData != ks.ExpectedData() {
			logger.Printf("%v does not match %v", foundData, ks.ExpectedData())

			// Only indicate that a reboot is required if the Key was marked
			// as we're considering a mismatch to be evidence. While unlikely,
			// we may wish to include Key values in our list that we are not
			// 100% certain indicate a need for a reboot.
			if ks.Evidence().DataOtherThanX {
				logger.Println("Reboot Required!")
				return restart.RebootCheckResult{
					Examined:       ks,
					RebootRequired: true,
					RebootReasons: []string{
						fmt.Sprintf(
							"Data for value %s for key %s found", ks.Value(), ks,
						),
					},
				}
			}
		}
	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary. If the earlier base Key
	// evaluation indicated that a reboot was needed, let's use that but with
	// a minor change to emphasize that this super type was examined.
	if baseKeyEvalResult.RebootRequired {
		result := baseKeyEvalResult
		result.Examined = ks

		return result
	}

	// Otherwise, fallback to a standard "no reboot required" result.
	return restart.RebootCheckResult{
		Examined:       ks,
		RebootRequired: false,
	}
}

// Data returns the actual data stored for a registry key value.
func (ks KeyStrings) Data() []string {
	return ks.data
}

// ExpectedData returns the expected data stored for a registry key value.
func (ks KeyStrings) ExpectedData() []string {
	return ks.expectedData
}

// DataDisplay provides a string representation of a registry key values's
// actual data for display purposes.
func (ks KeyStrings) DataDisplay() string {
	logger.Printf("Called for %+v", ks)
	return strings.Join(ks.Data(), ", ")
}

// AdditionalEvidence indicates what additional evidence "markers" have been
// supplied. If the reboot evidence markers for the Key type are not matched,
// these  (also optional) set of evidence markers are then checked to
// determine if a reboot is required.
func (ks KeyStrings) AdditionalEvidence() KeyStringsRebootEvidence {
	return ks.additionalEvidence
}

// Evaluate performs evaluation of the embedded Key value and then applies
// (optional) evaluation of the data field to determine whether any of the
// specified strings are found in the retrieved key value data. Any single
// match indicates a reboot is needed.
func (ks KeyStrings) Evaluate() restart.RebootCheckResult {

	// Evaluate embedded "base" Key first where we check shared requirements
	// and reboot evidence. We also explicitly indicate that we wish to retain
	// a handle to the open registry key (for use here).
	baseKeyEvalResult := ks.evaluate(false)

	defer func() {
		logger.Printf("Closing open handle to %q", ks)
		if err := ks.close(); err != nil {
			logger.Printf("Failed to close handle to open key %q", ks)
		}

		if ks.handle != nil {
			logger.Printf("Failed to close handle to %q", ks)
		}
		logger.Printf("Closed handle to %q", ks)
	}()

	// Go no further if an error occurred evaluating the "base" Key.
	if baseKeyEvalResult.Err != nil {
		return baseKeyEvalResult
	}

	if ks.Value() != "" {
		foundData, _, err := ks.handle.GetStringsValue(ks.Value())
		switch {
		case errors.Is(err, registry.ErrNotExist):
			if ks.Requirements().ValueRequired {
				logger.Printf("Value %q not found, but marked as required.", ks.Value())
				return restart.RebootCheckResult{
					Examined:       ks,
					RebootRequired: false,
					Err: fmt.Errorf(
						"value %s not found, but marked as required: %w",
						ks.Value(),
						restart.ErrMissingValue,
					),
				}
			}

			logger.Printf("Value %q not found, but not marked as required.", ks.Value())
			return restart.RebootCheckResult{
				Examined:       ks,
				RebootRequired: false,
			}

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

			return restart.RebootCheckResult{
				Examined:       ks,
				RebootRequired: false,
				Err: fmt.Errorf(
					"unexpected error occurred while retrieving %s value %s: %w",
					valReqLabel,
					ks.Value(),
					err,
				),
			}
		}

		logger.Printf("Data for value %q retrieved ...", ks.Value())
		logger.Printf("foundData: %v", foundData)
		logger.Printf("searchTerms: %v", ks.Data())
		logger.Print("Saving retrieved data for later use ...")
		ks.data = append(ks.data, foundData...)

		var valuesFound int
		for _, searchTerm := range ks.ExpectedData() {
			switch {
			case textutils.InList(searchTerm, foundData, true):
				valuesFound++

				logger.Printf("Found match %q within %v", searchTerm, ks.Data())

				// If we are just looking for one value, go ahead and return
				// early without checking for other matches.
				if ks.AdditionalEvidence().ValueFound {

					logger.Println("Reboot Required!")

					return restart.RebootCheckResult{
						Examined:       ks,
						RebootRequired: true,
						RebootReasons: []string{
							fmt.Sprintf(
								"Found match %s in data for value %s of key %s",
								searchTerm,
								ks.Value(),
								ks,
							),
						},
					}
				}

			default:
				logger.Printf("No matches found for %v", searchTerm)
			}
		}

		if ks.AdditionalEvidence().AllValuesFound {
			if valuesFound == len(ks.ExpectedData()) {
				// 100% match: All specified string values were found.
				return restart.RebootCheckResult{
					Examined:       ks,
					RebootRequired: true,
					RebootReasons: []string{
						fmt.Sprintf(
							"All specified strings found in data for value %s of key %s",
							ks.Value(),
							ks,
						),
					},
				}

			}
		}

	}

	// If we made it this far then nothing specific to this "super type"
	// indicated that a reboot was necessary. If the earlier base Key
	// evaluation indicated that a reboot was needed, let's use that but with
	// a minor change to emphasize that this super type was examined.
	if baseKeyEvalResult.RebootRequired {
		result := baseKeyEvalResult
		result.Examined = ks

		return result
	}

	// Otherwise, fallback to a standard "no reboot required" result.
	return restart.RebootCheckResult{
		Examined:       ks,
		RebootRequired: false,
	}

}

// Data returns the actual data stores for both registry key values.
func (kp KeyPair) Data() [][]byte {
	return kp.data
}

// DataDisplay provides a string representation of the data for both registry
// key values.
func (kp KeyPair) DataDisplay() string {

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
func (kp KeyPair) AdditionalEvidence() KeyPairRebootEvidence {
	return kp.additionalEvidence
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
func (kp KeyPair) Evaluate() restart.RebootCheckResult {

	foundData := make([][]byte, 0, 2)

	for _, key := range kp.Keys {

		// Evaluate embedded "base" Key first where we check shared
		// requirements and reboot evidence. We also explicitly indicate that
		// we wish to retain a handle to the open registry key (for use here).
		baseKeyEvalResult := key.evaluate(false)

		defer func() {
			logger.Printf("Closing open handle to %q", key)
			if err := key.close(); err != nil {
				logger.Printf("Failed to close handle to open key %q", key)
			}

			if key.handle != nil {
				logger.Printf("Failed to close handle to %q", key)
			}
			logger.Printf("Closed handle to %q", key)
		}()

		// Go no further if an error occurred evaluating the "base" Key.
		if baseKeyEvalResult.Err != nil {
			return baseKeyEvalResult
		}

		// Go no further if an error occurred evaluating the "base" Key or if
		// that evaluation was sufficient to determine a reboot is needed.
		//
		// Unlike other Key* types, this type is not meant to collect the
		// actual data registry values for later display purposes, only make
		// comparisons if requested. So, unlike the other Key* types, it is
		// sufficient to use only the base Key evaluation results if a reboot
		// was found to be required or an error occurred.
		if baseKeyEvalResult.Err != nil || baseKeyEvalResult.RebootRequired {
			return baseKeyEvalResult
		}

		// Exit early if we are not evaluating whether the data for each
		// registry key value matches.
		if !kp.AdditionalEvidence().PairedValuesDoNotMatch {
			return restart.RebootCheckResult{
				Examined:       kp,
				RebootRequired: false,
			}
		}

		if key.Value() != "" {
			bufSize, valType, err := key.handle.GetValue(key.Value(), nil)
			switch {
			case errors.Is(err, registry.ErrNotExist):
				if key.Requirements().ValueRequired {
					logger.Printf("Value %q not found, but marked as required.", key.Value())
					return restart.RebootCheckResult{
						// TODO: Should we report that we examined the current key
						// we are evaluating or the KeyPair instead?
						Examined:       kp,
						RebootRequired: false,
						Err: fmt.Errorf(
							"value %s not found, but marked as required: %w",
							key.Value(),
							restart.ErrMissingValue,
						),
					}
				}

				logger.Printf("Value %q not found, but not marked as required.", key.Value())
				return restart.RebootCheckResult{
					// TODO: Should we report that we examined the current key we
					// are evaluating or the KeyPair instead?
					Examined:       kp,
					RebootRequired: false,
				}

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

				return restart.RebootCheckResult{
					// TODO: Should we report that we examined the current key we
					// are evaluating or the KeyPair instead?
					Examined:       kp,
					RebootRequired: false,
					Err: fmt.Errorf(
						"unexpected error occurred while retrieving %s value %s: %w",
						valReqLabel,
						key.Value(),
						err,
					),
				}
			}

			logger.Printf(
				"Required buffer size %d for value %v of type %v ...",
				bufSize,
				key.Value(),
				getValueType(valType),
			)

			buffer := make([]byte, bufSize)
			_, _, err = key.handle.GetValue(key.Value(), buffer)

			// We intentionally use simpler error handling here since we just
			// evaluated whether the registry key value is required to be
			// present and have successfully retrieved the necessary buffer
			// size for the data associated with registry key value; if an
			// error occurs at this point it doesn't really matter what
			// requirement is being applied.
			//
			// TODO: Worth the complexity to implement a retry?
			if err != nil {
				return restart.RebootCheckResult{
					// TODO: Should we report that we examined the current key we
					// are evaluating or the KeyPair instead?
					Examined:       kp,
					RebootRequired: false,
					Err: fmt.Errorf(
						"failed to retrieve data for value %s: %w",
						key.Value(),
						err,
					),
				}

			}
			logger.Printf("data in raw/hex format: % x", buffer)
			logger.Printf("data in string format: %s", buffer)
			foundData = append(foundData, buffer)

		}
	}

	// If data was collected, evaluate it.
	if len(foundData) > 0 {

		// Make a copy of the retrieved data for later use.
		kp.data = make([][]byte, 2)

		logger.Print("Saving retrieved data for later use ...")
		copied := copy(kp.data, foundData)

		if copied != len(foundData) {
			panic(fmt.Sprintf(
				"Failed to copy all of foundData; "+
					"want %d elements, copied %d elements",
				len(foundData),
				copied,
			))
		}

		// compare retrieved data values
		fqpath1 := fmt.Sprintf(`%s\%s`, kp.Keys[0].Path(), kp.Keys[0].Value())
		fqpath2 := fmt.Sprintf(`%s\%s`, kp.Keys[1].Path(), kp.Keys[1].Value())

		if !bytes.Equal(foundData[0], foundData[1]) {
			logger.Printf("Data for %q does not equal %q", fqpath1, fqpath2)
			logger.Println("Reboot Required!")

			return restart.RebootCheckResult{
				Examined:       kp,
				RebootRequired: true,
				RebootReasons: []string{
					fmt.Sprintf(
						"Data mismatch for %s and %s",
						fqpath1,
						fqpath2,
					),
				},
			}
		}

		logger.Printf("Data equal for %q and %q", fqpath1, fqpath2)

	}

	// If we made it this far then nothing specific to this enclosing type
	// indicated that a reboot was necessary. Because this is not a "super
	// type" that embeds the base Key type, we cannot use a base Key
	// evaluation as a fallback state and so we explicitly indicate that per
	// this evaluation no reboot is required.
	return restart.RebootCheckResult{
		Examined:       kp,
		RebootRequired: false,
	}

}

// Validate performs basic validation. An error is returned for any validation
// failures.
func (kp KeyPair) Validate() error {

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

func (kp KeyPair) String() string {

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
