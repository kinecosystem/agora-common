package timeutil

import (
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

var regex = regexp.MustCompile(`(?i)([-+]?)P(?:([-+]?[0-9]+)D)?(T(?:([-+]?[0-9]+)H)?(?:([-+]?[0-9]+)M)?(?:([-+]?[0-9]+)(?:[.,]([0-9]{0,9}))?S)?)?`)

func addDuration(x, y time.Duration) (time.Duration, error) {
	r := x + y

	if (x^r)&(y^r) < 0 {
		return 0, errors.New("time.Duration overflow")
	}

	return r, nil
}

func absInt64(i int64) int64 {
	if i >= 0 {
		return i
	}
	return -i
}

func IsISO8601(s string) bool {
	return regex.MatchString(s)
}

// Parses an ISO-8601 formatted duration.
//
// This attempts to mimic Java's Duration.parse. Note the leading plus/minus
// sign, and negative values for other units are not part of the ISO-8601
// standard but Java includes them so we handle them here as well.
func ParseISO8601(s string) (duration time.Duration, err error) {
	orig := s

	if !IsISO8601(s) {
		return 0, errors.New("invalid duration " + orig)
	}

	matches := regex.FindStringSubmatch(s)

	negate := 1
	if "-" == matches[1] {
		negate = -1
	}
	dayMatch := matches[2]
	hourMatch := matches[4]
	minuteMatch := matches[5]
	secondMatch := matches[6]
	nanoMatchPadded := matches[7] + "000000000"
	nanoMatch := nanoMatchPadded[0:9]

	if dayMatch == "" && hourMatch == "" && minuteMatch == "" && secondMatch == "" {
		return 0, errors.New("invalid duration " + orig)
	}

	var days, hours, minutes, seconds, nanos int64

	if dayMatch != "" {
		days, err = strconv.ParseInt(dayMatch, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "invalid duration "+orig)
		}

		if absInt64(days) > (1<<63-1)/int64(time.Hour*24) {
			return 0, errors.New("invalid duration " + orig)
		}
	}

	if hourMatch != "" {
		hours, err = strconv.ParseInt(hourMatch, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "invalid duration "+orig)
		}

		if absInt64(hours) > (1<<63-1)/int64(time.Hour) {
			return 0, errors.New("invalid duration " + orig)
		}
	}

	if minuteMatch != "" {
		minutes, err = strconv.ParseInt(minuteMatch, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "invalid duration "+orig)
		}

		if absInt64(minutes) > (1<<63-1)/int64(time.Minute) {
			return 0, errors.New("invalid duration " + orig)
		}
	}

	if secondMatch != "" {
		seconds, err = strconv.ParseInt(secondMatch, 10, 64)
		if err != nil {
			return 0, errors.Wrap(err, "invalid duration "+orig)
		}

		if absInt64(seconds) > (1<<63-1)/int64(time.Second) {
			return 0, errors.New("invalid duration " + orig)
		}
	}

	if nanoMatch != "" {
		nanos, err = strconv.ParseInt(nanoMatch, 10, 64)
		if err != nil {
			// regex is [0-9]{0,9} which is always a valid int so this should never happen
			return 0, errors.Wrap(err, "invalid duration "+orig)
		}
	}

	duration = time.Duration(days) * time.Hour * 24

	duration, err = addDuration(duration, time.Duration(hours)*time.Hour)
	if err != nil {
		return 0, errors.Wrap(err, "invalid duration "+orig)
	}

	duration, err = addDuration(duration, time.Duration(minutes)*time.Minute)
	if err != nil {
		return 0, errors.Wrap(err, "invalid duration "+orig)
	}

	duration, err = addDuration(duration, time.Duration(seconds)*time.Second)
	if err != nil {
		return 0, errors.Wrap(err, "invalid duration "+orig)
	}

	duration, err = addDuration(duration, time.Duration(nanos)*time.Nanosecond)
	if err != nil {
		return 0, errors.Wrap(err, "invalid duration "+orig)
	}

	return time.Duration(negate) * duration, nil
}
