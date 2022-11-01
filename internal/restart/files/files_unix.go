//go:build !windows
// +build !windows

// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package files

import (
	"github.com/atc0005/check-restart/internal/restart"
)

// DefaultRebootRequiredIgnoredPaths provides the default collection of paths
// for registry related reboot required assertions that should be ignored.
func DefaultRebootRequiredIgnoredPaths() []string {
	return []string{}
}

// DefaultRebootRequiredAssertions provides the default collection of file
// related reboot required assertions.
func DefaultRebootRequiredAssertions() restart.RebootRequiredAsserters {

	// TODO: Look for paths specific to non-Windows, UNIX-like systems that
	// indicate a need for a system reboot.
	return restart.RebootRequiredAsserters{}
}
