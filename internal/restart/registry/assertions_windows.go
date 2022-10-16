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

// DefaultRebootRequiredAssertions provides the default collection of registry
// related reboot required assertions.
func DefaultRebootRequiredAssertions() restart.RebootRequiredAsserters {

	var assertions = restart.RebootRequiredAsserters{
		KeyInt{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SOFTWARE\Microsoft\Updates`,
				value: "UpdateExeVolatile",
				evidence: KeyRebootEvidence{
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
		KeyStrings{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SYSTEM\CurrentControlSet\Control\Session Manager`,
				value: "PendingFileRenameOperations",
				evidence: KeyRebootEvidence{
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
		KeyStrings{
			Key: Key{
				root:  registry.LOCAL_MACHINE,
				path:  `SYSTEM\CurrentControlSet\Control\Session Manager`,
				value: "PendingFileRenameOperations2",
				evidence: KeyRebootEvidence{
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
		Key{
			root: registry.LOCAL_MACHINE,

			// When a reboot is needed this key exists and contains one or
			// more REG_DWORD values with data set to 0x00000001; the
			// existence of the key is sufficient to indicate a reboot is
			// needed.
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,

			// When a reboot is needed there are subkeys. Observed subkeys
			// have a GUID naming pattern.
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Services\Pending`,
			evidence: KeyRebootEvidence{
				SubKeysExist: true,
			},

			requirements: KeyAssertions{
				KeyRequired: false,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\PostRebootReporting`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SOFTWARE\Microsoft\Windows\CurrentVersion\RunOnce`,
			value: "DVDRebootSignal",
			evidence: KeyRebootEvidence{
				ValueExists: true,
			},
			requirements: KeyAssertions{
				KeyRequired: true,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootInProgress`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\PackagesPending`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root: registry.LOCAL_MACHINE,
			path: `SOFTWARE\Microsoft\ServerManager\CurrentRebootAttempts`,
			evidence: KeyRebootEvidence{
				KeyExists: true,
			},
		},
		Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SYSTEM\CurrentControlSet\Services\Netlogon`,
			value: "JoinDomain",
			evidence: KeyRebootEvidence{
				ValueExists: true,
			},
		},
		Key{
			root:  registry.LOCAL_MACHINE,
			path:  `SYSTEM\CurrentControlSet\Services\Netlogon`,
			value: "AvoidSpnSet",
			evidence: KeyRebootEvidence{
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
		KeyPair{
			additionalEvidence: KeyPairRebootEvidence{
				PairedValuesDoNotMatch: true,
			},
			Keys: Keys{
				Key{
					root:  registry.LOCAL_MACHINE,
					path:  `SYSTEM\CurrentControlSet\Control\ComputerName\ActiveComputerName`,
					value: "ComputerName",
					requirements: KeyAssertions{
						KeyRequired:   true,
						ValueRequired: true,
					},
				},
				Key{
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
