<!-- omit in toc -->
# check-restart

Go-based tooling used to detect whether a restart (service) or reboot (system) is needed.

[![Latest Release](https://img.shields.io/github/release/atc0005/check-restart.svg?style=flat-square)](https://github.com/atc0005/check-restart/releases/latest)
[![Go Reference](https://pkg.go.dev/badge/github.com/atc0005/check-restart.svg)](https://pkg.go.dev/github.com/atc0005/check-restart)
[![go.mod Go version](https://img.shields.io/github/go-mod/go-version/atc0005/check-restart)](https://github.com/atc0005/check-restart)
[![Lint and Build](https://github.com/atc0005/check-restart/actions/workflows/lint-and-build.yml/badge.svg)](https://github.com/atc0005/check-restart/actions/workflows/lint-and-build.yml)
[![Project Analysis](https://github.com/atc0005/check-restart/actions/workflows/project-analysis.yml/badge.svg)](https://github.com/atc0005/check-restart/actions/workflows/project-analysis.yml)
[![Push Validation](https://github.com/atc0005/check-restart/actions/workflows/push-validation.yml/badge.svg)](https://github.com/atc0005/check-restart/actions/workflows/push-validation.yml)

<!-- omit in toc -->
## Table of Contents

- [Project home](#project-home)
- [Overview](#overview)
- [Features](#features)
- [Changelog](#changelog)
- [Requirements](#requirements)
  - [Building source code](#building-source-code)
  - [Running](#running)
- [Installation](#installation)
  - [From source](#from-source)
  - [Using release binaries](#using-release-binaries)
  - [Deployment](#deployment)
- [Configuration options](#configuration-options)
- [Configuration](#configuration)
  - [Command-line arguments](#command-line-arguments)
    - [`check_reboot`](#check_reboot)
  - [Logging output](#logging-output)
- [Examples](#examples)
  - [`OK` result](#ok-result)
  - [`WARNING` result](#warning-result)
    - [Without `verbose` flag and with logging enabled](#without-verbose-flag-and-with-logging-enabled)
    - [Verbose output without logging](#verbose-output-without-logging)
  - [`CRITICAL` result](#critical-result)
- [License](#license)
- [References](#references)

## Project home

See [our GitHub repo][repo-url] for the latest code, to file an issue or
submit improvements for review and potential inclusion into the project.

## Overview

This repo is intended to provide various tools used to detect whether a
restart (service) or reboot (system) is needed.

| Tool Name      | Overall Status | Description                                                                 |
| -------------- | -------------- | --------------------------------------------------------------------------- |
| `check_reboot` | Alpha          | Nagios plugin used to monitor for "reboot needed" status of Windows systems |

## Features

- Nagios plugin (`check_reboot`) for monitoring "reboot needed" status of
  Windows systems
  - NOTE: The intent is to support multiple operating systems, but as of this
    writing Windows is the only supported OS

- Optional branding "signature"
  - used to indicate what Nagios plugin (and what version) is responsible for
    the service check result

- Optional, leveled logging using `rs/zerolog` package
  - JSON-format output (to `stderr`)
  - choice of `disabled`, `panic`, `fatal`, `error`, `warn`, `info` (the
    default), `debug` or `trace`.

## Changelog

See the [`CHANGELOG.md`](CHANGELOG.md) file for the changes associated with
each release of this application. Changes that have been merged to `master`,
but not yet an official release may also be noted in the file under the
`Unreleased` section. A helpful link to the Git commit history since the last
official release is also provided for further review.

## Requirements

The following is a loose guideline. Other combinations of Go and operating
systems for building and running tools from this repo may work, but have not
been tested.

### Building source code

- Go
  - see this project's `go.mod` file for *preferred* version
  - this project tests against [officially supported Go
    releases][go-supported-releases]
    - the most recent stable release (aka, "stable")
    - the prior, but still supported release (aka, "oldstable")
- GCC
  - if building with custom options (as the provided `Makefile` does)
- `make`
  - if using the provided `Makefile`

### Running

- Windows 8.1
- Windows 10

- Windows Server 2012R2
- Windows Server 2016
- Windows Server 2022

## Installation

### From source

1. [Download][go-docs-download] Go
1. [Install][go-docs-install] Go
1. Clone the repo
   1. `cd /tmp`
   1. `git clone https://github.com/atc0005/check-restart`
   1. `cd check-restart`
1. Install dependencies (optional)
   - for Ubuntu Linux
     - `sudo apt-get install make gcc`
   - for CentOS Linux
     1. `sudo yum install make gcc`
1. Build
   - manually, explicitly specifying target OS and architecture
     - `GOOS=windows GOARCH=amd64 go build -mod=vendor ./cmd/check_reboot/`
       - most likely this is what you want
       - substitute `amd64` with the appropriate architecture if using
         different hardware (e.g., `arm64`)
   - using Makefile `windows` recipe
     - `make windows`
       - generates x86 and x64 binaries
   - using Makefile `release-build` recipe
     - `make release-build`
       - generates the same release assets as provided by this project's
         releases
1. Locate generated binaries
   - if using `Makefile`
     - look in `/tmp/check-restart/release_assets/check_reboot/`
   - if using `go build`
     - look in `/tmp/check-restart/`
1. Copy the applicable binaries to whatever systems needs to run them so that
   they can be deployed

### Using release binaries

1. Download the [latest release][repo-url] binaries
1. Copy the applicable binaries to whatever systems needs to run them so that
   they can be deployed

### Deployment

1. Place `check_reboot` in a location where it can be executed by the
   monitoring agent
   - Usually the same place as other Nagios plugins
   - For example, on a default Windows system with `NSClient++ x64` installed
    the `check_reboot.exe` plugin would be deployed to `C:\Program
    Files\NSClient++\scripts\custom\check_reboot.exe`
1. Update the monitoring agent configuration configuration (on a system that
   you wish to monitor for "needs reboot" condition) to create a new command
   definition
   - We will pretend that the DNS name for the system is `windows.example.com`

   - ```ini
     [/settings/external scripts/scripts]
     ; NOTE: stderr output is returned mixed in with stdout content. Disable logging to prevent this.
     ; check_reboot=scripts\\custom\\check_reboot.exe --verbose --log-level disabled
     ;
     check_reboot=scripts\\custom\\check_reboot.exe --verbose
     ```

   - see [NSClient++ External scripts doc][nsclient-external-scripts] for
     additional details
1. Restart the `nscp` service (label of `NSClient++ (x64)`)
1. Create a new Nagios "console" command definition to allow requesting the
   monitoring agent to run the plugin

   - ```nagios
     define command {
         # $ARG1$ is an optional non-default port
         command_name    wcheck_reboot
         command_line    $USER1$/check_nrpe -H $HOSTADDRESS$ -p $ARG1$ -t 30 -c check_reboot
     }
     ```

1. Create a new Nagios "console" service check that requests the monitoring
   agent to execute the plugin

   - ```nagios
     define service {
         host_name              windows.example.com
         service_description    check reboot
         use                    generic-service
         check_command          wcheck_reboot!12345!!!!!!!
         notification_period    24x7withMaintenanceWindow
         contacts               atc0005
         register               1
     }
     ```

## Configuration options

## Configuration

### Command-line arguments

- Use the `-h` or `--help` flag to display current usage information.
- Flags marked as **`required`** must be set via CLI flag.
- Flags *not* marked as required are for settings where a useful default is
  already defined, but may be overridden if desired.

#### `check_reboot`

| Flag              | Required | Default | Repeat | Possible                                                                | Description                                                                                          |
| ----------------- | -------- | ------- | ------ | ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| `branding`        | No       | `false` | No     | `branding`                                                              | Toggles emission of branding details with plugin status details. This output is disabled by default. |
| `h`, `help`       | No       | `false` | No     | `h`, `help`                                                             | Show Help text along with the list of supported flags.                                               |
| `version`         | No       | `false` | No     | `version`                                                               | Whether to display application version and then immediately exit application.                        |
| `v`, `verbose`    | No       | `false` | No     | `v`, `verbose`                                                          | Toggles emission of detailed output. This level of output is disabled by default.                    |
| `ll`, `log-level` | No       | `info`  | No     | `disabled`, `panic`, `fatal`, `error`, `warn`, `info`, `debug`, `trace` | Log message priority filter. Log messages with a lower level are ignored.                            |

### Logging output

Early testing using NSClient++ suggests that both `stderr` and `stdout` are
mixed together and sent back to the Nagios console as one output stream.

If this becomes a problem you can disable logging output (`--log-level
disabled`) to prevent `stderr` from being including in the output sent back to
the Nagios console.

If you know of a way to prevent `stderr` from being mixed into the output sent
back to Nagios, please file a GitHub issue in this project to share your
findings.

## Examples

### `OK` result

No reboot needed.

This output is emitted by the plugin when a reboot is not needed.

```console
C:\Users\Administrator>"C:\Program Files\NSClient++\scripts\custom\check_reboot.exe" --verbose
OK: Reboot not needed (applied 15 reboot assertions, 0 matched)


Summary:

  - 15 total reboot assertions applied
  - 0 total reboot assertions matched

--------------------------------------------------

Reboot not required

 | 'registry_assertions'=14;;;; 'assertions_matched'=0;;;; 'errors'=0;;;; 'time'=0ms;;;; 'all_assertions'=15;;;; 'file_assertions'=1;;;;
```

Regarding the output:

- The last line beginning with a space and the `|` symbol are performance
  data metrics emitted by the plugin. Depending on your monitoring system, these
  metrics may be collected and exposed as graphs/charts.
- This output was captured on a Windows 10 system, but is comparable to the
  output emitted by other Windows desktop & server systems.

### `WARNING` result

This output is emitted by the plugin when a reboot is needed.

#### Without `verbose` flag and with logging enabled

```console
C:\Users\Administrator>"C:\Program Files\NSClient++\scripts\custom\check_reboot.exe"
4:16AM ERR T:/github/check-restart/cmd/check_reboot/main.go:193 > Reboot assertions matched, reboot needed app_type=plugin logging_level=info num_reboot_assertions_applied=15 num_reboot_assertions_matched=5 version="check-restart x.y.z (https://github.com/atc0005/check-restart)"
WARNING: Reboot needed (applied 15 reboot assertions, 5 matched)

**ERRORS**

* reboot assertions matched, reboot needed

**DETAILED INFO**


Summary:

  - 15 total reboot assertions applied
  - 5 total reboot assertions matched

--------------------------------------------------

Reboot required because:


  - Value PendingFileRenameOperations of type MULTI_SZ for key HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager found

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired found

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending found

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\PackagesPending found

  - File C:\Windows\WinSxS\pending.xml found


 | 'assertions_matched'=5;;;; 'errors'=0;;;; 'time'=4ms;;;; 'all_assertions'=15;;;; 'file_assertions'=1;;;; 'registry_assertions'=14;;;;
```

Regarding the output:

- The last line beginning with a space and the `|` symbol are performance
  data metrics emitted by the plugin. Depending on your monitoring system, these
  metrics may be collected and exposed as graphs/charts.
- The first line is emitted to `stderr`. Where the other output is intended
  for use by Nagios to collect and display (via web UI or notifications), this
  output is intended for humans to directly read when troubleshooting plugin
  execution. If desired, this output can be muted by way of the `disabled`
  option for the `log-level` flag. See [Logging output](#logging-output) for
  additional details.
- This output was captured on a Windows Server 2022 system, but is comparable
  to the output emitted by other Windows desktop & server systems.

#### Verbose output without logging

```console
C:\Users\Administrator>"C:\Program Files\NSClient++\scripts\custom\check_reboot.exe" --verbose --log-level disabled
WARNING: Reboot needed (applied 15 reboot assertions, 5 matched)

**ERRORS**

* reboot assertions matched, reboot needed

**DETAILED INFO**


Summary:

  - 15 total reboot assertions applied
  - 5 total reboot assertions matched

--------------------------------------------------

Reboot required because:


  - Value PendingFileRenameOperations of type MULTI_SZ for key HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager found
    \??\C:\Program Files (x86)\Microsoft\EdgeUpdate\1.3.167.21,

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\WindowsUpdate\Auto Update\RebootRequired found

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\RebootPending found

  - Key HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Component Based Servicing\PackagesPending found

  - File C:\Windows\WinSxS\pending.xml found


 | 'assertions_matched'=5;;;; 'errors'=0;;;; 'time'=4ms;;;; 'all_assertions'=15;;;; 'file_assertions'=1;;;; 'registry_assertions'=14;;;;
```

Regarding the output:

- The last line beginning with a space and the `|` symbol are performance
  data metrics emitted by the plugin. Depending on your monitoring system, these
  metrics may be collected and exposed as graphs/charts.
- The first line is emitted to `stderr`. Where the other output is intended
  for use by Nagios to collect and display (via web UI or notifications), this
  output is intended for humans to directly read when troubleshooting plugin
  execution. If desired, this output can be muted by way of the `disabled`
  option for the `log-level` flag. See [Logging output](#logging-output) for
  additional details.
- This output was captured on a Windows Server 2022 system, but is comparable
  to the output emitted by other Windows desktop & server systems.

### `CRITICAL` result

This result is returned when an error occurs during the attempt to determine
whether a reboot is needed.

TODO: Provide example output when this scenario is encountered.

## License

See the [LICENSE](LICENSE) file for details.

## References

- <https://adamtheautomator.com/pending-reboot-registry/>
- <https://github.com/adbertram/Random-PowerShell-Work/blob/master/Random%20Stuff/Test-PendingReboot.ps1>

<!-- Footnotes here  -->

[repo-url]: <https://github.com/atc0005/check-restart>  "This project's GitHub repo"

[go-docs-download]: <https://golang.org/dl>  "Download Go"

[go-docs-install]: <https://golang.org/doc/install>  "Install Go"

[go-supported-releases]: <https://go.dev/doc/devel/release#policy> "Go Release Policy"

[nsclient-external-scripts]: <https://docs.nsclient.org/howto/external_scripts/> "NSClient++ External scripts"
