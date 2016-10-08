// Package semver answers the question 'is this version newer than
// that version', according to the semantic versioning specification:
// http://semver.org/
//
// Expected use is:
//    v1Newer, err := semver.Version(version1).IsNewerThan(version2)
package semver

import (
	"github.com/blang/semver"
	"github.com/pkg/errors"
)

// semver.Version().IsNewerThan()

// ParsedVersion represents something we can compare to another
// version.
type ParsedVersion struct {
	v   semver.Version
	err error
}

// Version parses a version.  If there is an error from parsing, it
// won't be available until you call version.IsNewerThan().
func Version(s string) ParsedVersion {
	v, err := semver.ParseTolerant(s)
	return ParsedVersion{v, errors.Wrapf(err, "Unable to parse %q", s)}
}

// IsNewerThan returns true if pv is newer than s.  An error will be
// returned if there was a problem parsing either version.
func (pv ParsedVersion) IsNewerThan(s string) (bool, error) {
	if pv.err != nil {
		return false, pv.err
	}
	v, err := semver.ParseTolerant(s)
	if err != nil {
		return false, errors.Wrapf(err, "Unable to parse %q", s)
	}
	return pv.v.GT(v), nil
}
