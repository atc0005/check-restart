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

[Unreleased]: https://github.com/atc0005/check-cert/compare/v0.1.1...HEAD
[v0.1.0]: https://github.com/atc0005/check-cert/releases/tag/v0.1.0
[v0.1.1]: https://github.com/atc0005/check-cert/releases/tag/v0.1.1
