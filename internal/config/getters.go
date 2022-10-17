// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package config

// supportedLogLevels returns a list of valid log levels supported by tools in
// this project.
func supportedLogLevels() []string {
	return []string{
		LogLevelDisabled,
		LogLevelPanic,
		LogLevelFatal,
		LogLevelError,
		LogLevelWarn,
		LogLevelInfo,
		LogLevelDebug,
		LogLevelTrace,
	}
}
