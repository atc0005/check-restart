# Changelog

## Overview

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Please [open an issue](https://github.com/atc0005/check-restart/issues) for any
deviations that you spot; I'm still learning!.

## Types of changes

The following types of changes will be recorded in this file:

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Removed` for now removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

## [Unreleased]

- placeholder

## [v0.5.2] - 2023-07-14

### Overview

- RPM package improvements
- Bug fixes
- Dependency updates
- built using Go 1.19.11
  - Statically linked
  - Windows (x86, x64)
  - Linux (x86, x64)

### Changed

- Dependencies
  - `Go`
    - `1.19.10` to `1.19.11`
  - `atc0005/go-nagios`
    - `v0.15.0` to `v0.16.0`
  - `atc0005/go-ci`
    - `go-ci-oldstable-build-v0.11.0` to `go-ci-oldstable-build-v0.11.4`
  - `golang.org/x/sys`
    - `v0.8.0` to `v0.10.0`
- (GH-163) Update RPM postinstall scripts to use restorecon
- (GH-154) Update vuln analysis GHAW to remove on.push hook

### Fixed

- (GH-156) Restore local CodeQL workflow
- (GH-161) Correct logging format listed in README

## [v0.5.1] - 2023-06-08

### Overview

- Bug fixes
- Pattern/Path match improvements & workarounds
- Dependency updates
- built using Go 1.19.10
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `Go`
    - `1.19.9` to `1.19.10`
  - `atc0005/go-nagios`
    - `v0.14.0` to `v0.15.0`
  - `atc0005/go-ci` build image
    - `go-ci-oldstable-build-v0.10.5` to `go-ci-oldstable-build-v0.10.6`
  - `mattn/go-isatty`
    - `v0.0.18` to `v0.0.19`
- (GH-146) Normalize matched/ignored paths
- (GH-151) Temporarily disable problematic assertions

### Fixed

- (GH-148) Disable depguard linter

## [v0.5.0] - 2023-05-11

### Overview

- Build improvements
- Bug fixes
- Dependency updates
- built using Go 1.19.9
  - Statically linked
  - Windows (x64)

### Added

- (GH-132) Add rootless container builds via Docker/Podman

### Changed

- Dependencies
  - `Go`
    - `1.19.7` to `1.19.9`
  - `atc0005/go-ci` build image
    - `go-ci-oldstable-build-v0.9.2` to `go-ci-oldstable-build-v0.10.5`
  - `rs/zerolog`
    - `v1.29.0` to `v1.29.1`
  - `golang.org/x/sys`
    - `v0.6.0` to `v0.8.0`
  - `mattn/go-isatty`
    - `v0.0.17` to `v0.0.18`

### Fixed

- (GH-125) Update vuln analysis GHAW to use on.push hook
- (GH-141) Fix revive linting error

## [v0.4.0] - 2023-03-17

### Overview

- Output tweaks
- Documentation refresh
- built using Go 1.19.7
  - Statically linked
  - Windows (x64)

### Changed

- (GH-116) Cleanup & sample REG_MULTI_SZ reg key values
- (GH-118) Reduce sampling of MULTI_SZ key values further
- (GH-119) Reduce plugin output further by omitting Summary section
- (GH-121) List total/omitted entries count for `REG_MULTI_SZ` registry key
  values
- (GH-123) Update README to reflect recent output tweaks

### Fixed

- (GH-115) Verbose "reboot evidence" for PendingFileRenameOperations
  REG_MULTI_SZ registry key is *too* verbose resulting in potential loss of
  additional reboot required reasons

## [v0.3.0] - 2023-03-09

### Overview

- Add support for generating packages
- Generated binary changes
  - filename patterns
  - compression
  - executable metadata
- Build improvements
- built using Go 1.19.7
  - Statically linked
  - Windows (x64)

### Added

- (GH-95) Generate RPM/DEB packages using nFPM

### Changed

- (GH-94) Switch to semantic versioning (semver) compatible versioning
  pattern
- (GH-96) Add version metadata to Windows executables
- (GH-97) Makefile: Compress binaries and use fixed filenames
- (GH-98) Makefile: Refresh recipes to add "standard" set, new
  package-related options
- (GH-99) Build dev/stable releases using go-ci Docker image

## [v0.2.3] - 2023-03-08

### Overview

- Dependency updates
- built using Go 1.19.7
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `Go`
    - `1.19.4` to `1.19.7`
  - `atc0005/go-nagios`
    - `v0.10.2` to `v0.14.0`
  - `rs/zerolog`
    - `v1.28.0` to `v1.29.0`
  - `golang.org/x/sys`
    - `v0.3.0` to `v0.6.0`
  - `mattn/go-isatty`
    - `v0.0.16` to `v0.0.17`
- (GH-71) Drop explicit plugin runtime tracking
  - allow the new nagios package functionality to handle tracking and emitting
    the time metric automatically at plugin completion
- (GH-73) Update nagios library usage to reflect dep changes
- GitHub Actions Workflows
  - (GH-79) Add Go Module Validation, Dependency Updates jobs
  - (GH-87) Drop `Push Validation` workflow
  - (GH-88) Rework workflow scheduling
  - (GH-92) Remove `Push Validation` workflow status badge
- Builds
  - (GH-80) Add missing Makefile usage entry for release build
  - (GH-81) Add missing "clean" step to recipe

### Fixed

- (GH-69) Fix mispelling of Inspector app type
- (GH-75) Add missing copyright header to test file
- (GH-80) Add missing Makefile usage entry for release build
- (GH-81) Add missing "clean" step to recipe
- (GH-89) Listed registry key paths are stripped of separators
- (GH-101) Listed `MULTI_SZ` registry key paths are stripped of separators
- (GH-104) Use UNKNOWN state for invalid command-line args
- (GH-105) Use UNKNOWN state for perfdata add failure
- (GH-106) Use UNKNOWN state for failure to validate assertions

## [v0.2.2] - 2022-12-09

### Overview

- Dependency updates
- built using Go 1.19.4
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `Go`
    - `1.19.3` to `1.19.4`

## [v0.2.1] - 2022-12-05

### Overview

- Bug fixes
- Dependency updates
- built using Go 1.19.3
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `Go`
    - `1.19.2` to `1.19.3`
  - `golang.org/x/sys`
    - `v0.1.0` to `v0.3.0`

### Fixed

- (GH-53) README: Remove duplicate "Configuration" header
- (GH-56) Explicitly disable colorized plugin logger output
- (GH-57) Enable pkg debug logging if global Debug or Trace
- (GH-58) Fix project name in CHANGELOG links
- (GH-59) Minor refactor of perfdata handling
- (GH-60) Add doc comments for interface methods
- (GH-61) Reduce complexity of reports.writeAssertions func
- (GH-63) Resolve gocognit linter warnings

## [v0.2.0] - 2022-11-01

### Overview

- Add default set of ignored path entries (optionally disable)
- Minor polish
- built using Go 1.19.2
  - Statically linked
  - Windows (x64)

### Added

- (GH-32) Add default set of ignored path entries
- (GH-47) Add support for disabling set of default ignored path entries

### Changed

- (GH-44) Mute "reboot assertions matched, reboot needed" message by default

## [v0.1.3] - 2022-10-24

### Overview

- Bug fixes
- built using Go 1.19.2
  - Statically linked
  - Windows (x64)

### Changed

- (GH-37) Rename "assertions" performance data metrics to clarify meaning
- (GH-38) Temporarily disable problematic registry path

## [v0.1.2] - 2022-10-19

### Overview

- Dependency updates
- built using Go 1.19.2
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `atc0005/go-nagios`
    - `v0.10.1` to `v0.10.2`

### Fixed

- (GH-29) Add (retroactively) an `Overview` section to CHANGELOG entries

## [v0.1.1] - 2022-10-18

### Overview

- Bug fixes
- Dependency updates
- built using Go 1.19.2
  - Statically linked
  - Windows (x64)

### Changed

- Dependencies
  - `golang.org/x/sys`
    - `v0.0.0-20221013171732-95e765b1cc43` to `v0.1.0`
- (GH-25) Update `release-build` Makefile recipe
- (GH-27) Update README installation directions

### Fixed

- (GH-21) Issues with `config.supportedLogLevels()` helper function
- (GH-24) Add missing section to CHANGELOG

## [v0.1.0] - 2022-10-17

### Overview

- Initial release
- built using Go 1.19.2
  - Statically linked
  - Windows (x86, x64)

### Added

Initial release!

This release provides an early release version of a Nagios plugin used to
monitor for "reboot needed" status of Windows systems. Tested on multiple
Windows desktop and server variants.

[Unreleased]: https://github.com/atc0005/check-restart/compare/v0.5.2...HEAD
[v0.5.2]: https://github.com/atc0005/check-restart/releases/tag/v0.5.2
[v0.5.1]: https://github.com/atc0005/check-restart/releases/tag/v0.5.1
[v0.5.0]: https://github.com/atc0005/check-restart/releases/tag/v0.5.0
[v0.4.0]: https://github.com/atc0005/check-restart/releases/tag/v0.4.0
[v0.3.0]: https://github.com/atc0005/check-restart/releases/tag/v0.3.0
[v0.2.3]: https://github.com/atc0005/check-restart/releases/tag/v0.2.3
[v0.2.2]: https://github.com/atc0005/check-restart/releases/tag/v0.2.2
[v0.2.1]: https://github.com/atc0005/check-restart/releases/tag/v0.2.1
[v0.2.0]: https://github.com/atc0005/check-restart/releases/tag/v0.2.0
[v0.1.3]: https://github.com/atc0005/check-restart/releases/tag/v0.1.3
[v0.1.2]: https://github.com/atc0005/check-restart/releases/tag/v0.1.2
[v0.1.1]: https://github.com/atc0005/check-restart/releases/tag/v0.1.1
[v0.1.0]: https://github.com/atc0005/check-restart/releases/tag/v0.1.0
