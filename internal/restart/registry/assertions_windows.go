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
	"github.com/atc0005/check-restart/internal/restart"
	"golang.org/x/sys/windows/registry"
)

// DefaultRebootRequiredIgnoredPaths provides the default collection of paths
// for registry related reboot required assertions that should be ignored.
//
// Paths are normalized before comparison with matched paths.
//
// For consistency, these entries should match the default path syntax for the
// operating system in question.
func DefaultRebootRequiredIgnoredPaths() []string {
	return []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Services\Pending\117cab2d-82b1-4b5a-a08c-4d62dbee7782`,
	}
}

// DefaultRebootRequiredAssertions provides the default collection of registry
// related reboot required assertions.
func DefaultRebootRequiredAssertions() restart.RebootRequiredAsserters {

	var assertions = restart.RebootRequiredAsserters{
		&KeyInt{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SOFTWARE\Microsoft\Updates`,
				value: "UpdateExeVolatile",
				evidenceExpected: KeyRebootEvidence{
					// TODO: Is there a valid scenario where this would be
					// false, yet we're specifying data for a registry key
					// value?
					//
					// One potential scenario might be if we're not confident
					// of a specific value being a reboot indicator and we
					// just want to log that a mismatch occurs for further
					// consideration.
					DataOtherThanX: true,
				},
			},
			expectedData: 0,
		},
		&KeyStrings{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SYSTEM\CurrentControlSet\Control\Session Manager`,
				value: "PendingFileRenameOperations",
				evidenceExpected: KeyRebootEvidence{
					// FIXME: Based on recent experience, this is a VERY noisy
					// evidence marker. Just having the value present has not
					// proven sufficient to indicate the need for a reboot.
					ValueExists: true,
				},
			},

			// TODO: this is the default and not really needed. Adding this
			// just for the time being as a reminder that the support is
			// available.
			additionalEvidence: KeyStringsRebootEvidence{
				ValueFound:     false,
				AllValuesFound: false,
			},
		},
		&KeyStrings{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SYSTEM\CurrentControlSet\Control\Session Manager`,
				value: "PendingFileRenameOperations2",
				evidenceExpected: KeyRebootEvidence{
					ValueExists: true,
				},
			},

			// TODO: this is the default and not really needed. Adding this
			// just for the time being as a reminder that the support is
			// available.
			additionalEvidence: KeyStringsRebootEvidence{
				ValueFound:     false,
				AllValuesFound: false,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,

			// When a reboot is needed this key exists and contains one or
			// more REG_DWORD values with data set to 0x00000001; the
			// existence of the key is sufficient to indicate a reboot is
			// needed.
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,

			// When a reboot is needed there are subkeys. Observed subkeys
			// have a GUID naming pattern.
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Services\Pending`,
			evidenceExpected: KeyRebootEvidence{
				SubKeysExist: true,
			},

			requirements: KeyAssertions{
				KeyRequired: false,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\PostRebootReporting`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SOFTWARE\Microsoft\Windows\CurrentVersion\RunOnce`,
			value: "DVDRebootSignal",
			evidenceExpected: KeyRebootEvidence{
				ValueExists: true,
			},
			requirements: KeyAssertions{
				KeyRequired: true,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootInProgress`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\PackagesPending`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\ServerManager\CurrentRebootAttempts`,
			evidenceExpected: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		&Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SYSTEM\CurrentControlSet\Services\Netlogon`,
			value: "JoinDomain",
			evidenceExpected: KeyRebootEvidence{
				ValueExists: true,
			},
		},
		&Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SYSTEM\CurrentControlSet\Services\Netlogon`,
			value: "AvoidSpnSet",
			evidenceExpected: KeyRebootEvidence{
				ValueExists: true,
			},
		},

		// The intent with the KeyPair type is to support key pairs that are
		// completely optional.
		//
		// In this case, the KeyPair is non-optional as both key paths are
		// expected to be present on all supported Windows versions.
		//
		// Here we explicitly note that non-matching key value data indicates
		// a reboot AND that both the key and value are required for the
		// specific data that we're comparing.
		&KeyPair{
			additionalEvidence: KeyPairRebootEvidence{
				PairedValuesDoNotMatch: true,
			},
			Keys: Keys{
				&Key{
					root:  registry.LOCAL_MACHINE,
					path:  `SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName`,
					value: "ComputerName",
					requirements: KeyAssertions{
						KeyRequired:   true,
						ValueRequired: true,
					},
				},
				&Key{
					root:  registry.LOCAL_MACHINE,
					path:  `SYSTEM\CurrentControlSet\Control\ComputerName\ComputerName`,
					value: "ComputerName",
					requirements: KeyAssertions{
						KeyRequired:   true,
						ValueRequired: true,
					},
				},
			},
		},
	}

	return assertions

}
