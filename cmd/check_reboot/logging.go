// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package main

import (
	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/check-restart/internal/restart/files"
	"github.com/atc0005/check-restart/internal/restart/registry"
	"github.com/atc0005/check-restart/internal/restart/reports"
	"github.com/rs/zerolog"
)

func handleLibraryLogging() {
	switch {
	case zerolog.GlobalLevel() == zerolog.DebugLevel ||
		zerolog.GlobalLevel() == zerolog.TraceLevel:
		restart.EnableLogging()
		files.EnableLogging()
		registry.EnableLogging()
		reports.EnableLogging()
	default:
		restart.DisableLogging()
		files.DisableLogging()
		registry.DisableLogging()
		reports.DisableLogging()
	}
}
