package alpm

import "unicode"

// Note: this is modeled very closely on version.c in order to make it
// easy to compare the two implementations.

type parsed struct {
	epoch   []rune
	version []rune
	release []rune
}

func parseEVR(text string) parsed {
	epoch := []rune{}
	version := []rune{}
	release := []rune{}

	evr := []rune(text)
	l := len(evr)

	var s int // points to epoch terminator
	se := -1  // points to version terminator
	for i := l - 1; i >= 0; i-- {
		if evr[i] == '-' {
			se = i
			break
		}
	}

	for s = 0; s+1 < l && unicode.IsDigit(evr[s]); s++ {
	}

	var versionStart int
	if evr[s] == ':' {
		epoch = evr[:s]
		versionStart = s + 1
		if len(epoch) == 0 {
			epoch = []rune("0")
		}
	} else {
		epoch = []rune("0")
		versionStart = 0
	}
	if se != -1 {
		release = evr[se+1:]
		version = evr[versionStart:se]
	} else {
		version = evr[versionStart:]
	}

	return parsed{epoch, version, release}
}

type tokenType int

const (
	sepType     tokenType = iota
	alphaType             // segment composed of letters
	numericType           // segment composed of digits
)

type token struct {
	kind tokenType
	text []rune
}

func (t token) stripLeadingZeros() []rune {
	for i, r := range t.text {
		if r != '0' {
			return t.text[i:]
		}
	}
	return t.text[0:0]
}

func runeKind(r rune) tokenType {
	switch {
	case unicode.IsLetter(r):
		return alphaType
	case unicode.IsDigit(r):
		return numericType
	default:
		return sepType
	}
}

type separator int

func nextSeparator(source []rune) (sep separator, remaining []rune) {
	if len(source) == 0 {
		return 0, source
	}
	for i, r := range source {
		if runeKind(r) != sepType {
			return separator(i), source[i:]
		}
	}
	return separator(len(source)), source[0:0]
}

func (s separator) len() int {
	return int(s)
}

func nextToken(source []rune) (t token, remaining []rune) {
	if len(source) == 0 {
		return token{sepType, source}, source
	}
	kind := runeKind(source[0])
	for i, r := range source {
		k := runeKind(r)
		if k != kind {
			return token{kind, source[:i]}, source[i:]
		}
	}
	return token{kind, source}, source[0:0]
}

// like C strcmp, but for slices of runes
func runeCmp(one, two []rune) int {
	oneLen := len(one)
	twoLen := len(two)
	i := 0
	for {
		if i >= oneLen && i >= twoLen {
			// We have compared all runes and both runs ended at the
			// same time.
			return 0
		}
		if i >= oneLen || i >= twoLen {
			if twoLen > oneLen {
				return -1
			}
			return 1
		}

		rOne := one[i]
		rTwo := two[i]
		if rOne < rTwo {
			return -1
		}
		if rTwo < rOne {
			return 1
		}

		i++
	}
}

// rpmVerCmp compares epoch or version or release, deciding which is
// newer, similar to rpmvercmp in version.c.
//   0 if a and b are equal
//   1 if a is newer
//  -1 if b is newer
//
// We expect to find segments (runs of letters or runs of digits)
// possibly separated by separators, which are characters that are not
// letters or digits.
//
// When comparing separators, we just compare length, with the longer
// one being newer.
//
// Digit segments are always newer than letter segments or empty
// segments.
//
// When comparing letter segments, we do the equivalent of C strcmp.
//
// When comparing digit segments, we strip leading zeros.  If the
// lengths are different, then the longest one is newer.  Otherwise
// treat as letter segment.
//
// If we have looped through all segments, we have special logic for
// the final pair:
//	 if a is empty and b is numeric, b is newer.
//	 if a is an alpha, b is newer.
//	 otherwise a is newer.
func rpmVerCmp(a, b []rune) int {
	for {
		var sepA, sepB separator
		sepA, a = nextSeparator(a)
		sepB, b = nextSeparator(b)

		// If we ran to the end of either, we are finished with the loop
		if len(a) == 0 || len(b) == 0 {
			break
		}

		// If the separator lengths were different, we are also finished
		if sepA.len() != sepB.len() {
			if sepA.len() < sepB.len() {
				return -1
			}
			return 1
		}

		var tokA, tokB token
		tokA, a = nextToken(a)
		tokB, b = nextToken(b)

		// this cannot happen, as we previously tested to make sure that
		// the first string has a non-null segment
		if tokA.kind == sepType {
			return -1 // arbitrary
		}

		// take care of the case where the two version segments are
		// different types: one numeric, the other alpha.  Numeric
		// segments are always newer than alpha segments XXX See patch
		// #60884 (and details) from bugzilla #50977.
		if tokA.kind != tokB.kind {
			if tokA.kind == numericType {
				return 1
			}
			return -1
		}

		if tokA.kind == numericType {
			// this used to be done by converting the digit segments
			// to ints using atoi() - it's changed because long
			// digit segments can overflow an int - this should fix that.

			// throw away any leading zeros - it's a number, right?
			tokA.text = tokA.stripLeadingZeros()
			tokB.text = tokB.stripLeadingZeros()

			// whichever number has more digits wins
			if len(tokA.text) < len(tokB.text) {
				return -1
			}
			if len(tokB.text) < len(tokA.text) {
				return 1
			}
		}

		rc := runeCmp(tokA.text, tokB.text)
		if rc != 0 {
			return rc
		}
	}

	// This catches the case where all numeric and alpha segments have
	// compared identically and we ran out of more segments.
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	/* the final showdown. we never want a remaining alpha string to
	 * beat an empty string. the logic is a bit weird, but:
	 * - if a is empty and b is not an alpha, b is newer.
	 * - if a is an alpha, b is newer.
	 * - otherwise a is newer.
	 * */
	var tokA, tokB token
	tokA, _ = nextToken(a)
	tokB, _ = nextToken(b)
	if (tokA.kind == sepType && tokB.kind != alphaType) || tokA.kind == alphaType {
		return -1
	}
	return 1
}

// VerCmp compares two Arch linux package version strings.  It returns:
//   0 if the versions are equal
//  -1 if a is older
//   1 if a is newer
//
// This is the go equivalent of the C function alpm_pkg_vercmp.
//
// Note that this is not semantic versioning.
//
// Based on the contents of pacman/lib/libalpm/version.c, version
// v5.0.1.  This was apparently equivalent to rpmvercmp, in rpm
// version 4.8.1.
//
// Different epoch values for version strings will override any further
// comparison. If no epoch is provided, 0 is assumed.
//
// Keep in mind that the pkgrel is only compared if it is available
// on both versions handed to this function. For example, comparing
// 1.5-1 and 1.5 will yield 0; comparing 1.5-1 and 1.5-2 will yield
// -1 as expected. This is mainly for supporting versioned dependencies
// that do not include the pkgrel.
func VerCmp(a, b string) int {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	if len(a) == 0 {
		return -1
	}
	if len(b) == 0 {
		return 1
	}
	if a == b {
		return 0
	}

	parsedA := parseEVR(a)
	parsedB := parseEVR(b)

	if ret := rpmVerCmp(parsedA.epoch, parsedB.epoch); ret != 0 {
		return ret
	}
	if ret := rpmVerCmp(parsedA.version, parsedB.version); ret != 0 {
		return ret
	}
	if len(parsedA.release) > 0 && len(parsedB.release) > 0 {
		return rpmVerCmp(parsedA.release, parsedB.release)
	}

	return 0
}

// Less compares a to be as arch linux package version strings and
// returns true if a is less than (older than) b.
func Less(a, b string) bool {
	return VerCmp(a, b) == -1
}
