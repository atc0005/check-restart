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

[Unreleased]: https://github.com/atc0005/check-restart/compare/v0.2.2...HEAD
[v0.2.2]: https://github.com/atc0005/check-restart/releases/tag/v0.2.2
[v0.2.1]: https://github.com/atc0005/check-restart/releases/tag/v0.2.1
[v0.2.0]: https://github.com/atc0005/check-restart/releases/tag/v0.2.0
[v0.1.3]: https://github.com/atc0005/check-restart/releases/tag/v0.1.3
[v0.1.2]: https://github.com/atc0005/check-restart/releases/tag/v0.1.2
[v0.1.1]: https://github.com/atc0005/check-restart/releases/tag/v0.1.1
[v0.1.0]: https://github.com/atc0005/check-restart/releases/tag/v0.1.0
