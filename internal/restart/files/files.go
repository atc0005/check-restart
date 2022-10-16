// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package files

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/atc0005/check-restart/internal/restart"
)

// Add an "implements assertion" to fail the build if the
// restart.RebootRequiredAsserter implementation isn't correct.
var _ restart.RebootRequiredAsserter = (*File)(nil)

// Add an "implements assertion" to fail the build if the
// restart.FileRebootRequired implementation isn't correct.
var _ FileRebootRequired = (*File)(nil)

// FileRebootRequired represents the behavior of a file that can be evaluated
// to indicate whether a reboot is required.
//
// TODO: This is not needed at present, but would be useful later if/when
// adding support for optional files or when evaluating a file's metadata or
// contents (e.g., a specific value for a specific line in a file).
type FileRebootRequired interface {
	Validate() error
	Path() string
	Requirements() FileAssertions
	String() string
}

// FileRebootEvidence indicates what file evidence is required in order to
// determine that a reboot is needed.
type FileRebootEvidence struct {
	FileExists     bool
	FileEmpty      bool
	FileNotEmpty   bool
	FileExecutable bool
	FileIsSymlink  bool
}

// FileAssertions indicates what requirements must be met. If not met, this
// indicates than an error has occurred. If a specific file is required, but
// not present on a system then client code can not reliably determine whether
// a reboot is necessary. IN that scenario client code should assume that all
// results are invalid.
//
// TODO: This is not needed at present, but would be useful later if/when
// adding support for evaluating a file's metadata or contents (e.g., a
// specific value for a specific line in a file).
type FileAssertions struct {
	FileRequired bool
}

// File represents a file that if found (and requirements met) indicates a
// reboot is needed.
//
// TODO: At present, just finding the file is sufficient to indicate a reboot.
// An enclosing type could be added to apply more specific requirements (e.g.,
// such as finding a specific value on a specific line in a file).
type File struct {
	// path is either the fully-qualified path to a file or, if
	// envVarPathPrefix is set is a partial path to be joined to
	// envVarPathPrefix to form a fully-qualified path to a file.
	path string

	// envVarPathPrefix if set, will be prepended to path to form the
	// fully-qualified path to a file.
	envVarPathPrefix string

	// evidence indicates what is required in order to determine that a reboot
	// is needed.
	// evidence FileRebootEvidence

	// requirements indicates what requirements must be met. If not met, this
	// indicates that an error has occurred.
	requirements FileAssertions
}

func (f File) Validate() error {

	if f.path == "" {
		return fmt.Errorf(
			"invalid file path: %w",
			restart.ErrMissingValue,
		)
	}

	return nil

}

func (f File) Path() string {
	return f.path
}

// Requirements returns the specified requirements or file assertions. If one
// of these requirements is not met than an error condition has been
// encountered. Requirements does not indicate whether a reboot is needed,
// only how potential "not found" conditions should be treated.
func (f File) Requirements() FileAssertions {
	return f.requirements
}

// String implements the Stringer interface and provides the fully qualified
// path to a file. If the specified environment variable is found that value
// is prepended to the given path value to form the fully qualified path to
// the file. If an environment variable is not specified, the given path value
// is expected to be fully qualified.
func (f File) String() string {

	var pathPrefix string
	if f.envVarPathPrefix != "" {
		pathPrefix = os.Getenv(f.envVarPathPrefix)
	}

	switch {
	case pathPrefix != "":
		return filepath.Join(pathPrefix, f.path)
	default:
		return f.path
	}

}

func (f File) Evaluate() restart.RebootCheckResult {

	logger.Printf("Given file: %s", f)

	filePath := filepath.Clean(f.String())
	logger.Printf("File after sanitizing path: %s", filePath)

	_, err := os.Stat(filePath)
	switch {
	case os.IsNotExist(err):
		logger.Printf("File %s not found, reboot not required due to this file.", filePath)
		return restart.RebootCheckResult{
			Examined:       f,
			RebootRequired: false,
		}

	case err != nil:
		return restart.RebootCheckResult{
			Examined:       f,
			RebootRequired: false,
			Err: fmt.Errorf(
				"unexpected error occurred while opening file %s: %v",
				filePath,
				err,
			),
		}

	default:
		logger.Printf("File %q found!", filePath)
		logger.Println("Reboot Required!")
		return restart.RebootCheckResult{
			Examined:       f,
			RebootRequired: true,
			RebootReasons: []string{
				fmt.Sprintf(
					"File %s found", filePath,
				),
			},
		}
	}

}
