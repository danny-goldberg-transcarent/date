// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package date

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// MustAutoParse is as per AutoParse except that it panics if the string cannot be parsed.
// This is intended for setup code; don't use it for user inputs.
func MustAutoParse(value string) Date {
	d, err := AutoParse(value)
	if err != nil {
		panic(err)
	}
	return d
}

// MustAutoParseUS is as per AutoParseUS except that it panics if the string cannot be parsed.
// This is intended for setup code; don't use it for user inputs.
func MustAutoParseUS(value string) Date {
	d, err := AutoParseUS(value)
	if err != nil {
		panic(err)
	}
	return d
}

// AutoParse is like ParseISO, except that it automatically adapts to a variety of date formats
// provided that they can be detected unambiguously. Specifically, this includes the widely-used
// "European" and "British" date formats but not the common US format. Surrounding whitespace is
// ignored.
//
// The supported formats are:
//
// * all formats supported by ParseISO
//
// * yyyy/mm/dd | yyyy.mm.dd (or any similar pattern)
//
// * dd/mm/yyyy | dd.mm.yyyy (or any similar pattern)
//
// * d/m/yyyy | d.m.yyyy (or any similar pattern)
//
// * surrounding whitespace is ignored
func AutoParse(value string) (Date, error) {
	return autoParse(value, func(yyyy, f1, f2 string) string { return fmt.Sprintf("%s-%s-%s", yyyy, f1, f2) })
}

// AutoParseUS is like ParseISO, except that it automatically adapts to a variety of date formats
// provided that they can be detected unambiguously. Specifically, this includes the widely-used
// "European" and "US" date formats but not the common "British" format. Surrounding whitespace is
// ignored.
//
// The supported formats are:
//
// * all formats supported by ParseISO
//
// * yyyy/mm/dd | yyyy.mm.dd (or any similar pattern)
//
// * mm/dd/yyyy | mm.dd.yyyy (or any similar pattern)
//
// * m/d/yyyy | m.d.yyyy (or any similar pattern)
//
// * surrounding whitespace is ignored
func AutoParseUS(value string) (Date, error) {
	return autoParse(value, func(yyyy, f1, f2 string) string { return fmt.Sprintf("%s-%s-%s", yyyy, f2, f1) })
}

func autoParse(value string, compose func(yyyy, f1, f2 string) string) (Date, error) {
	abs := strings.TrimSpace(value)
	if len(abs) == 0 {
		return 0, errors.New("Date.AutoParse: cannot parse a blank string")
	}

	sign := ""
	if abs[0] == '+' || abs[0] == '-' {
		sign = abs[:1]
		abs = abs[1:]
	}

	if len(abs) >= 8 {
		i1 := -1
		i2 := -1
		for i, r := range abs {
			if unicode.IsPunct(r) {
				if i1 < 0 {
					i1 = i
				} else {
					i2 = i
				}
			}
		}

		if i1 >= 4 && i2 > i1 && abs[i1] == abs[i2] {
			// just normalise the punctuation
			a := []byte(abs)
			a[i1] = '-'
			a[i2] = '-'
			abs = string(a)

		} else if i1 >= 1 && i2 > i1 && abs[i1] == abs[i2] {
			// harder case - need to swap the field order
			f1 := abs[0:i1]      // day or month
			f2 := abs[i1+1 : i2] // month or day
			if len(f1) == 1 {
				f1 = "0" + f1
			}
			if len(f2) == 1 {
				f2 = "0" + f2
			}
			yyyy := abs[i2+1:]
			abs = compose(yyyy, f2, f1)
		}
	}
	return parseISO(value, sign+abs)
}

// MustParseISO is as per ParseISO except that it panics if the string cannot be parsed.
// This is intended for setup code; don't use it for user inputs.
func MustParseISO(value string) Date {
	d, err := ParseISO(value)
	if err != nil {
		panic(err)
	}
	return d
}

// ParseISO parses an ISO 8601 formatted string and returns the date value it represents.
// It accepts the following formats:
//
//   - the common formats ±YYYY-MM-DD and ±YYYYMMDD (e.g. 2006-01-02 and 20060102)
//   - the ordinal date representation ±YYYY-OOO (e.g. 2006-217)
//
// For common formats, ParseISO will accept dates with more year digits than the four-digit
// minimum. A leading plus '+' sign is allowed and ignored. Basic format (without '-'
// separators) is allowed.
//
// If a time field is present, it is ignored. For example, "2018-02-03T00:00:00Z" is parsed as
// 3rd February 2018.
//
// For ordinal dates, the extended format (including '-') is supported, but the basic format
// (without '-') is not supported because it could not be distinguished from the YYYYMMDD format.
//
// See also date.Parse, which can be used to parse date strings in other formats; however, it
// only accepts years represented with exactly four digits.
//
// Background: https://en.wikipedia.org/wiki/ISO_8601#Dates
// https://www.iso.org/obp/ui#iso:std:iso:8601:-1:ed-1:v1:en:term:3.1.3.1
func ParseISO(value string) (Date, error) {
	return parseISO(value, value)
}

func parseISO(input, value string) (Date, error) {
	abs := value
	sign := 1

	if len(value) > 0 {
		switch value[0] {
		case '+':
			abs = value[1:]
		case '-':
			abs = value[1:]
			sign = -1
		}
	}

	tee := strings.IndexByte(abs, 'T')
	if tee == 8 || tee == 10 {
		if !timeRegex1.MatchString(abs[tee:]) && !timeRegex2.MatchString(abs[tee:]) {
			return 0, fmt.Errorf("date.ParseISO: date-time %q: not a time", value)
		}
		abs = abs[:tee]
	}

	dash1 := strings.IndexByte(abs, '-')
	dash2 := strings.LastIndexByte(abs, '-')

	if dash1 < 0 {
		// parse YYYYMMDD (more Y digits are allowed)
		ln := len(abs)
		fm := ln - 4
		fd := ln - 2
		if fm < 0 || fd < 0 {
			return 0, fmt.Errorf("date.ParseISO: cannot parse %q: too short", input)
		}

		return parseYYYYMMDD(input, abs[:fm], abs[fm:fd], abs[fd:], sign)
	}

	if dash2 > dash1 {
		// parse YYYY-MM-DD (more Y digits are allowed)
		fy1 := dash1
		fm1 := dash1 + 1
		fm2 := dash2
		fd1 := dash2 + 1

		if abs[fm2] != '-' {
			return 0, fmt.Errorf("date.ParseISO: cannot parse %q: incorrect syntax for date yyyy-mm-dd", input)
		}

		return parseYYYYMMDD(input, abs[:fy1], abs[fm1:fm2], abs[fd1:], sign)
	}

	// parse YYYY-OOO (more Y digits are allowed)
	fy1 := dash1
	fo1 := dash1 + 1

	if len(abs) != fo1+3 {
		return 0, fmt.Errorf("date.ParseISO: cannot parse %q: incorrect length for ordinal date yyyy-ooo", input)
	}

	return parseYYYYOOO(input, abs[:fy1], abs[fo1:], sign)
}

func parseYYYYMMDD(input, yyyy, mm, dd string, sign int) (Date, error) {
	year, e1 := parseField(yyyy, "year", 4, -1)
	month, e2 := parseField(mm, "month", -1, 2)
	day, e3 := parseField(dd, "day", -1, 2)

	err := errors.Join(e1, e2, e3)
	if err != nil {
		return 0, fmt.Errorf("date.ParseISO: cannot parse %q: %w", input, err)
	}

	t := time.Date(sign*year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return encode(t), nil
}

func parseYYYYOOO(input, yyyy, ooo string, sign int) (Date, error) {
	year, e1 := parseField(yyyy, "year", 4, -1)
	ordinal, e2 := parseField(ooo, "ordinal", -1, 3)

	err := errors.Join(e1, e2)
	if err != nil {
		return 0, fmt.Errorf("date.ParseISO: cannot parse ordinal date %q: %w", input, err)
	}

	t := time.Date(sign*year, time.January, ordinal, 0, 0, 0, 0, time.UTC)

	return encode(t), nil
}

var (
	timeRegex1 = regexp.MustCompile("^T[0-9][0-9].[0-9][0-9].[0-9][0-9]")
	timeRegex2 = regexp.MustCompile("^T[0-9]{2,6}")
)

func parseField(field, name string, minLength, requiredLength int) (int, error) {
	if (minLength > 0 && len(field) < minLength) || (requiredLength > 0 && len(field) != requiredLength) {
		return 0, fmt.Errorf("%s has wrong length", name)
	}
	number, err := strconv.Atoi(field)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return number, nil
}

// MustParse is as per Parse except that it panics if the string cannot be parsed.
// This is intended for setup code; don't use it for user inputs.
func MustParse(layout, value string) Date {
	d, err := Parse(layout, value)
	if err != nil {
		panic(err)
	}
	return d
}

// Parse parses a formatted string of a known layout and returns the Date value it represents.
// The layout defines the format by showing how the reference date, defined
// to be
//
//	Monday, Jan 2, 2006
//
// would be interpreted if it were the value; it serves as an example of the
// input format. The same interpretation will then be made to the input string.
//
// This function actually uses time.Parse to parse the input and can use any
// layout accepted by time.Parse, but returns only the date part of the
// parsed Time value.
//
// This function cannot currently parse ISO 8601 strings that use the expanded
// year format; you should use date.ParseISO to parse those strings correctly.
// That is, it only accepts years represented with exactly four digits.
func Parse(layout, value string) (Date, error) {
	t, err := time.Parse(layout, value)
	if err != nil {
		return 0, err
	}
	return encode(t), nil
}
