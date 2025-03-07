// Copyright 2015 Rick Beton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// MustParse is as per Parse except that it panics if the string cannot be parsed.
// This is intended for setup code; don't use it for user inputs.
func MustParse(hms string) Clock {
	t, err := Parse(hms)
	if err != nil {
		panic(err)
	}
	return t
}

// Parse converts a string representation to a Clock. Acceptable representations
// are as per ISO-8601 - see https://en.wikipedia.org/wiki/ISO_8601#Times
//
// Also, conventional AM- and PM-based strings are parsed, such as "2am", "2:45pm".
// Remember that 12am is midnight and 12pm is noon.
func Parse(hms string) (clock Clock, err error) {
	if strings.HasSuffix(hms, "am") || strings.HasSuffix(hms, "AM") {
		return parseAmPm(hms, 0)
	} else if strings.HasSuffix(hms, "pm") || strings.HasSuffix(hms, "PM") {
		return parseAmPm(hms, 12)
	}
	return parseISO(hms)
}

func parseISO(hms string) (clock Clock, err error) {
	switch len(hms) {
	case 2: // HH
		return parseClockParts(hms, hms, "", "", "", 0, 0, 0)

	case 4: // HHMM
		return parseClockParts(hms, hms[:2], hms[2:], "", "", 0, 0, 0)

	case 5: // HH:MM
		if hms[2] != ':' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, hms[:2], hms[3:], "", "", 0, 0, 0)

	case 6: // HHMMSS
		return parseClockParts(hms, hms[:2], hms[2:4], hms[4:], "", 0, 0, 0)

	case 8: // HH:MM:SS
		if hms[2] != ':' || hms[5] != ':' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, hms[:2], hms[3:5], hms[6:], "", 0, 0, 0)

	case 9, 10, 11, 12, 13, 14, 15, 16, 17, 18: // HH:MM:SS.000...
		if hms[2] != ':' || hms[5] != ':' || hms[8] != '.' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, hms[:2], hms[3:5], hms[6:8], hms[9:], 9, 0, 0)
	}
	return 0, parseError(hms)
}

func parseAmPm(hms string, offset int) (clock Clock, err error) {
	n := len(hms)

	switch len(hms) {
	case 3: // Ham
		return parseClockParts(hms, "0"+hms[:1], "", "", "", 0, 12, offset)

	case 4: // HHam
		return parseClockParts(hms, hms[:2], "", "", "", 0, 12, offset)
	}

	colon := strings.IndexByte(hms, ':')
	if colon < 0 {
		return 0, parseError(hms)
	}

	h := hms[:colon]
	rest := hms[colon+1 : n-2]

	switch len(rest) {
	case 2: // MM
		return parseClockParts(hms, h, rest, "", "", 0, 12, offset)

	case 5: // MM:SS
		if rest[2] != ':' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, h, rest[:2], rest[3:], "", 0, 12, offset)

	case 6, 7: // MM:SS.0xm
		if rest[2] != ':' || rest[5] != '.' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, h, rest[:2], rest[3:5], rest[6:], 8, 12, offset)

	case 8: // MM:SS.00xm
		if rest[2] != ':' || rest[5] != '.' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, h, rest[:2], rest[3:5], rest[6:], 7, 12, offset)

	case 9: // MM:SS.000xm
		if rest[2] != ':' || rest[5] != '.' {
			return 0, parseError(hms)
		}
		return parseClockParts(hms, h, rest[:2], rest[3:5], rest[6:], 6, 12, offset)
	}
	return 0, parseError(hms)
}

func parseClockParts(input, hh, mm, ss, fracs string, zeros, mod, offset int) (clock Clock, err error) {
	h := 0
	m := 0
	s := 0
	ns := 0

	if hh != "" {
		h, err = strconv.Atoi(hh)
		if err != nil {
			return 0, parseError(input)
		}
	}

	if mod > 0 {
		h = h % mod
	}

	if mm != "" {
		m, err = strconv.Atoi(mm)
		if err != nil {
			return 0, parseError(input)
		}
	}

	if ss != "" {
		s, err = strconv.Atoi(ss)
		if err != nil {
			return 0, parseError(input)
		}
	}

	c := New(h+offset, m, s, 0)

	if fracs != "" {
		fracs = (fracs + strings.Repeat("0", zeros))[:9]
		ns, err = strconv.Atoi(fracs)
		if err != nil {
			return 0, parseError(input)
		}
		c += Clock(ns)
	}

	return c, nil
}

func parseError(hms string) error {
	_, _, line, _ := runtime.Caller(1)
	return fmt.Errorf("parse.go:%d: clock.Clock: cannot parse %q", line, hms)
}
