//go:build windows
// +build windows

// Copyright 2022 Adam Chalkley
//
// https://github.com/atc0005/check-restart
//
// Licensed under the MIT License. See LICENSE file in the project root for
// full license information.

package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/atc0005/check-restart/internal/restart"
	"github.com/atc0005/check-restart/internal/textutils"
)

// Add an "implements assertion" to fail the build if the
// restart.RebootRequiredAsserter implementation isn't correct.
var _ restart.RebootRequiredAsserter = (*File)(nil)

// Add an "implements assertion" to fail the build if the
// restart.FileRebootRequired implementation isn't correct.
var _ FileRebootRequired = (*File)(nil)

// Add "implements assertions" to fail the build if the restart.MatchedPath
// implementation isn't correct.
var _ restart.MatchedPath = (*MatchedPath)(nil)

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
	// Err() error // TODO: Should this be an interface method?
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

// FileRuntime is a collection of values for a File that are set during File
// evaluation. Unlike the static values set for a File (e.g., path, any
// requirements or assertions), these values are not known until execution or
// runtime.
type FileRuntime struct {
	// err records any error that occurs while performing an evaluation.
	err error

	// evidenceFound is the collection of evidence found when evaluating a
	// specified assertion.
	evidenceFound FileRebootEvidence

	// ignored indicates whether this value has been marked by filtering logic
	// as not considered when determining whether a reboot is needed.
	// ignored bool

	// pathsMatched is a collection of file path values that were matched
	// during evaluation of specified reboot required assertions.
	pathsMatched MatchedPathIndex
}

// MatchedPathIndex is a collection of path values that were matched during
// evaluation of specified reboot required assertions.
type MatchedPathIndex map[string]MatchedPath

// File represents a file that if found (and requirements met) indicates a
// reboot is needed.
type File struct {
	// path is either the fully-qualified path to a file or, if
	// envVarPathPrefix is set is a partial path to be joined to
	// envVarPathPrefix to form a fully-qualified path to a file.
	path string

	// envVarPathPrefix if set, will be prepended to path to form the
	// fully-qualified path to a file.
	envVarPathPrefix string

	// runtime is a collection of values that are set during evaluation.
	// Unlike static values that are known ahead of time, these values are not
	// known until execution or runtime.
	runtime FileRuntime

	// evidenceExpected indicates what evidence is used to determine that a
	// reboot is needed.
	evidenceExpected FileRebootEvidence

	// requirements indicates what requirements must be met. If not met, this
	// indicates that an error has occurred.
	requirements FileAssertions
}

// Err exposes the underlying error (if any) as-is.
func (f *File) Err() error {
	return f.runtime.err
}

// MatchedPath represents a path that was matched when performing an
// evaluation of a "reboot required" assertion.
type MatchedPath struct {
	// root is the left-most element of a matched path. This is the beginning
	// of a qualified path.
	root string

	// relative is the unqualified path (without the root element). The base
	// element of the path is usually included in this element.
	relative string

	// base is the last element or the right-most "leaf" value of a matched
	// path.
	base string

	// ignored indicates whether this value has been marked by filtering logic
	// as not considered when determining whether a reboot is needed.
	ignored bool
}

// Validate performs basic validation. An error is returned for any validation
// failures.
func (f *File) Validate() error {

	if f.path == "" {
		return fmt.Errorf(
			"invalid file path: %w",
			restart.ErrMissingValue,
		)
	}

	return nil

}

// Path returns the specified (potentially unqualified) path to the file.
func (f *File) Path() string {
	return f.path
}

// Requirements returns the specified requirements or file assertions. If one
// of these requirements is not met than an error condition has been
// encountered. Requirements does not indicate whether a reboot is needed,
// only how potential "not found" conditions should be treated.
func (f *File) Requirements() FileAssertions {
	return f.requirements
}

// String provides the fully qualified path for a File. If the specified
// environment variable is found that value is prepended to the given path
// value to form the fully qualified path to the file. If an environment
// variable is not specified, the given path value is expected to be fully
// qualified.
func (f *File) String() string {

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

// AddMatchedPath records given paths as successful assertion matches.
// Duplicate entries are ignored.
func (f *File) AddMatchedPath(paths ...string) {

	if f.runtime.pathsMatched == nil {
		f.runtime.pathsMatched = make(MatchedPathIndex)
	}

	for _, path := range paths {
		// Record MatchedPath if it does not already exist; we do not want to
		// overwrite an existing entry in case any non-default metadata is set
		// for the entry.
		if _, ok := f.runtime.pathsMatched[path]; !ok {

			// f.path may be unqualified. Unlike registry keys, these values are not
			// intentionally split out into "root" keys and path values.

			var rootPath string
			qualifiedPath, err := filepath.Abs(f.String())
			switch {
			case err != nil:
				rootPath = filepath.Dir(f.path)
			default:
				rootPath = filepath.Dir(qualifiedPath)
			}

			relPath, err := filepath.Rel(rootPath, path)
			switch {
			case err != nil:
				logger.Printf("Failed to obtain relative path for %q using %q as the base", path, rootPath)
				logger.Printf("Falling back to using %q as relative path", path)
				relPath = path
			default:
				logger.Printf(
					"Successfully resolved %q as relative path of %q using %q as root path",
					relPath,
					path,
					rootPath,
				)
			}

			matchedPath := MatchedPath{
				root:     rootPath,
				relative: relPath,
				base:     filepath.Base(path),
			}

			f.runtime.pathsMatched[path] = matchedPath
		}
	}

}

// MatchedPaths returns all recorded paths from successful assertion matches.
//
//	func (f *File) MatchedPaths() []string {
//		paths := make([]string, 0, len(f.runtime.pathsMatched))
//		for path := range f.runtime.pathsMatched {
//			paths = append(paths, path)
//		}
//		return paths
//	}
func (f *File) MatchedPaths() restart.MatchedPaths {

	pathStrings := make([]string, 0, len(f.runtime.pathsMatched))
	matchedPaths := make(restart.MatchedPaths, 0, len(f.runtime.pathsMatched))

	// Pull all of the keys.
	for k := range f.runtime.pathsMatched {
		pathStrings = append(pathStrings, k)
	}

	// Sort them.
	sort.Strings(sort.StringSlice(pathStrings))

	// Use them to pull out the MatchedPath entries in order.
	for _, path := range pathStrings {
		logger.Printf("File.runtime.pathsMatched entry: %q", path)
		matchedPaths = append(matchedPaths, f.runtime.pathsMatched[path])
	}

	return matchedPaths
}

// Evaluate applies the specified assertion to determine if a reboot is
// necessary.
func (f *File) Evaluate() {
	logger.Printf("Given file: %s", f)

	filePath := filepath.Clean(f.String())
	logger.Printf("File after sanitizing path: %s", filePath)

	_, err := os.Stat(filePath)
	switch {
	case os.IsNotExist(err):
		logger.Printf("File %s not found, reboot not required due to this file.", filePath)
		return

	case err != nil:
		f.runtime.err = err

		return

	default:
		logger.Printf("File %q found!", filePath)
		logger.Println("Reboot Required!")

		f.SetFoundEvidenceFileExists()
		f.AddMatchedPath(filePath)

		return
	}

}

// Filter uses the list of specified ignore patterns to mark each matched path
// for the File as ignored *IF* a match is found.
//
// While matched path and ignored pattern entries are normalized before
// comparison, we record path entries using the original non-normalized form.
//
// If no matched paths are recorded Filter makes no changes. Filter should be
// called before performing final state evaluation.
func (f *File) Filter(ignorePatterns []string) {

	numIgnorePatterns := len(ignorePatterns)
	var numIgnorePatternsApplied int

	if numIgnorePatterns == 0 {
		logger.Printf("0 ignore patterns specified for %q; skipping Filter", f)
		return
	}

	logger.Printf(
		"%d ignore patterns specified for %q; applying Filter",
		numIgnorePatterns,
		f,
	)

	for originalPathString, matchedPath := range f.runtime.pathsMatched {
		logger.Printf("Searching matched path %q for ignore pattern matches", originalPathString)

		normalizedPathString := textutils.NormalizePath(originalPathString)
		logger.Printf("Normalizing matched path %q as %q", originalPathString, normalizedPathString)

		for _, ignorePattern := range ignorePatterns {

			normalizedIgnorePattern := textutils.NormalizePath(ignorePattern)
			logger.Printf("Normalizing ignore pattern %q as %q", ignorePattern, normalizedIgnorePattern)

			if strings.Contains(normalizedPathString, normalizedIgnorePattern) {
				logger.Printf("matchedPath %q contains ignorePattern %q", originalPathString, ignorePattern)
				logger.Printf("marking matched path %q as ignored", originalPathString)

				matchedPath.ignored = true
				f.runtime.pathsMatched[originalPathString] = matchedPath
				numIgnorePatternsApplied++
			}
		}
	}

	logger.Printf("%d ignore patterns applied for %q", numIgnorePatternsApplied, f)
}

// ExpectedEvidence returns the specified evidence that (if found) indicates a
// reboot is needed.
func (f *File) ExpectedEvidence() FileRebootEvidence {
	return f.evidenceExpected
}

// DiscoveredEvidence returns the discovered evidence from an earlier
// evaluation.
func (f *File) DiscoveredEvidence() FileRebootEvidence {
	return f.runtime.evidenceFound
}

// SetFoundEvidenceFileExists records that the FileExists reboot evidence was
// found.
func (f *File) SetFoundEvidenceFileExists() {
	logger.Printf("Recording that the FileExists evidence was found for %q", f)
	f.runtime.evidenceFound.FileExists = true
}

// SetFoundEvidenceFileEmpty records that the FileEmpty reboot evidence was
// found.
func (f *File) SetFoundEvidenceFileEmpty() {
	logger.Printf("Recording that the FileEmpty evidence was found for %q", f)
	f.runtime.evidenceFound.FileEmpty = true
}

// SetFoundEvidenceFileNotEmpty records that the FileNotEmpty reboot evidence
// was found.
func (f *File) SetFoundEvidenceFileNotEmpty() {
	logger.Printf("Recording that the FileNotEmpty evidence was found for %q", f)
	f.runtime.evidenceFound.FileNotEmpty = true
}

// SetFoundEvidenceFileExecutable records that the FileExecutable reboot evidence
// was found.
func (f *File) SetFoundEvidenceFileExecutable() {
	logger.Printf("Recording that the FileExecutable evidence was found for %q", f)
	f.runtime.evidenceFound.FileExecutable = true
}

// SetFoundEvidenceFileIsSymlink records that the FileIsSymlink reboot evidence
// was found.
func (f *File) SetFoundEvidenceFileIsSymlink() {
	logger.Printf("Recording that the FileIsSymlink evidence was found for %q", f)
	f.runtime.evidenceFound.FileIsSymlink = true
}

// HasEvidence indicates whether any evidence was found for an assertion
// evaluation.
func (f *File) HasEvidence() bool {
	if f.runtime.evidenceFound.FileExists {
		return true
	}
	if f.runtime.evidenceFound.FileEmpty {
		return true
	}
	if f.runtime.evidenceFound.FileNotEmpty {
		return true
	}
	if f.runtime.evidenceFound.FileExecutable {
		return true
	}
	if f.runtime.evidenceFound.FileIsSymlink {
		return true
	}

	return false
}

// Ignored indicates whether the File has been marked as ignored.
func (f *File) Ignored() bool {

	numMatchedPaths := len(f.runtime.pathsMatched)

	// logger.Printf("%d pathsMatched entries for %q", numMatchedPaths, k)

	// An empty collection of entries can occur if an error occurred or if no
	// assertions were matched.
	if numMatchedPaths == 0 {
		// logger.Printf("%d pathsMatched entries for %q", numMatchedPaths, f)
		return false
	}

	for _, v := range f.runtime.pathsMatched {
		if !v.ignored {
			return false
		}

		logger.Printf("%s is marked as ignored\n", v)
	}

	// The entire File is ignored *only* if all recorded match path entries
	// are marked as ignored.
	return true
}

// HasIgnored indicates whether any matched path for the File have been marked
// as ignored.
func (f *File) HasIgnored() bool {
	for _, v := range f.runtime.pathsMatched {
		if v.ignored {
			return true
		}
	}

	return false
}

// RebootRequired indicates whether an evaluation determined that a reboot is
// needed. If the File has been marked as ignored (all recorded matched paths
// marked as ignored) the need for a reboot is not indicated.
func (f *File) RebootRequired() bool {
	if !f.Ignored() && f.HasEvidence() {
		return true
	}

	return false
}

// IsCriticalState indicates whether an evaluation determined that the File is
// in a CRITICAL state. Whether the File has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (f *File) IsCriticalState() bool {
	switch {

	// If we could determine that a reboot is required we consider that to be
	// a WARNING state.
	case !f.Ignored() && f.RebootRequired():
		return false

	// If we were unable to determine whether a reboot is required due to
	// errors we consider that to be a CRITICAL state.
	case !f.Ignored() && f.Err() != nil:
		if errors.Is(f.Err(), restart.ErrMissingOptionalItem) {
			return false
		}
		return true

	// No reboot required and no errors, not CRITICAL state.
	default:
		return false

	}
}

// IsWarningState indicates whether an evaluation determined that the File is
// in a WARNING state. Whether the File has been marked as Ignored is
// considered. The caller is responsible for filtering the collection prior to
// calling this method.
func (f *File) IsWarningState() bool {
	return !f.Ignored() && f.RebootRequired()
}

// IsOKState indicates whether an evaluation determined that the File is in an
// OK state. Whether the File has been marked as Ignored is considered. The
// caller is responsible for filtering the collection prior to calling this
// method.
//
// TODO: Cleanup the logic.
func (f *File) IsOKState() bool {
	switch {
	case f.Ignored():
		return true
	case !f.Ignored() && f.RebootRequired():
		return false
	case !f.Ignored() && f.Err() != nil:
		if errors.Is(f.Err(), restart.ErrMissingOptionalItem) {
			return true
		}
		return false
	default:
		return true
	}
}

// RebootReasons returns a list of the reasons associated with the evidence
// found for an evaluation that indicates a reboot is needed.
func (f *File) RebootReasons() []string {
	// The usual scenario is one reason per evidence match.
	reasons := make([]string, 0, 1)

	if f.runtime.evidenceFound.FileExists {
		reasons = append(reasons, fmt.Sprintf(
			"File %s found", f,
		))
	}

	if f.runtime.evidenceFound.FileEmpty {
		reasons = append(reasons, fmt.Sprintf(
			"File %s empty (but should not be)", f,
		))
	}

	if f.runtime.evidenceFound.FileNotEmpty {
		reasons = append(reasons, fmt.Sprintf(
			"File %s not empty (but expected to be)", f,
		))
	}

	if f.runtime.evidenceFound.FileExecutable {
		reasons = append(reasons, fmt.Sprintf(
			"File %s executable (but should not be)", f,
		))
	}

	if f.runtime.evidenceFound.FileIsSymlink {
		reasons = append(reasons, fmt.Sprintf(
			"File %s is a symbolic link (but should not be)", f,
		))
	}

	return reasons
}

// Root returns the left-most element of a matched path. This returned value
// is the beginning of a qualified path.
func (mp MatchedPath) Root() string {
	return mp.root
}

// Rel returns the relative (unqualified) element of a matched path. The base
// element of the path is usually included in this element.
func (mp MatchedPath) Rel() string {
	return mp.relative
}

// Base returns the last or right-most "leaf" element of a matched path.
func (mp MatchedPath) Base() string {
	return mp.base
}

// Full returns the qualified matched path value.
func (mp MatchedPath) Full() string {
	return filepath.Join(mp.root, mp.relative)
}

// String provides a human readable version of the matched path value.
func (mp MatchedPath) String() string {
	return mp.Full()
}

// func matchedPathsFromPathStrings(rootPath string, pathStrings []string) restart.MatchedPaths {
//
// 	matchedPaths := make(restart.MatchedPaths, 0, len(pathStrings))
//
// 	for _, path := range pathStrings {
// 		relPath, err := filepath.Rel(rootPath, path)
// 		switch {
// 		case err != nil:
// 			logger.Printf("Failed to obtain relative path for %q using %q as the base", path, rootPath)
// 			logger.Printf("Falling back to using %q as relative path", path)
// 			relPath = path
// 		default:
// 			logger.Printf(
// 				"Successfully resolved %q as relative path of %q using %q as root path",
// 				relPath,
// 				path,
// 				rootPath,
// 			)
// 		}
//
// 		matchedPath := MatchedPath{
// 			root:     rootPath,
// 			relative: relPath,
// 			base:     filepath.Base(path),
// 		}
//
// 		matchedPaths = append(matchedPaths, matchedPath)
// 	}
//
// 	return matchedPaths
//
// }
//
// func matchedPathFromPathString(rootPath string, pathString string) restart.MatchedPath {
//
// 	relPath, err := filepath.Rel(rootPath, pathString)
// 	switch {
// 	case err != nil:
// 		logger.Printf("Failed to obtain relative path for %q using %q as the base", pathString, rootPath)
// 		logger.Printf("Falling back to using %q as relative path", pathString)
// 		relPath = pathString
// 	default:
// 		logger.Printf(
// 			"Successfully resolved %q as relative path of %q using %q as root path",
// 			relPath,
// 			pathString,
// 			rootPath,
// 		)
// 	}
//
// 	matchedPath := MatchedPath{
// 		root:     rootPath,
// 		relative: relPath,
// 		base:     filepath.Base(pathString),
// 	}
//
// 	return matchedPath
// }
