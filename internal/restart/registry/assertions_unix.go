//go:build !windows
// +build !windows

// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package registry

// NOTE: This package is not intended for non-Windows systems.

import (
	"github.com/atc0005/check-restart/internal/restart"
)

// DefaultRebootRequiredIgnoredPaths provides the default collection of paths
// for registry related reboot required assertions that should be ignored.
func DefaultRebootRequiredIgnoredPaths() []string {

	logger.Println("WARNING: This tool is not supported for non-Windows systems!")
	return []string{}
}

// DefaultRebootRequiredAssertions provides the default collection of registry
// related reboot required assertions.
func DefaultRebootRequiredAssertions() restart.RebootRequiredAsserters {

	logger.Println("WARNING: This tool is not supported for non-Windows systems!")
	return restart.RebootRequiredAsserters{}
}
