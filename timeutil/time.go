package timeutil

import (
	"math"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	// MaxTimestampSeconds is the seconds field of the latest valid protobuf
	// timestamp. Note that the maximum possible timestamp also has
	// `MaxTimestampNanos` as the nanoseconds field.
	// This is time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC).Unix() - 1.
	// Reference: https://github.com/golang/protobuf/blob/master/ptypes/timestamp.go
	MaxTimestampSeconds = 253402300799

	// MinTimestampSeconds is the seconds field of the oldest valid protobuf
	// timestamp. This is equal to the seconds value of the zero time.
	// Reference: https://github.com/golang/protobuf/blob/master/ptypes/timestamp.go
	MinTimestampSeconds = -62135596800

	// MaxTimestampNanos is the largest number of nanoseconds allowed in a
	// protobuf timestamp. This is strictly less than a second.
	// Reference: https://github.com/golang/protobuf/blob/master/ptypes/timestamp.go#L73
	MaxTimestampNanos int64 = 1e9 - 1

	// maxNanoYear is the upper bound for a valid `time.UnixNano()`.
	// Everything past `time.Date(2262, 1, 1, 0, 0, 0, 0, time.UTC)` should
	// be treated as unsafe when calling `UnixNano` as per the docs.
	maxNanoYear = 2262

	// The phrase "Not all sec values have a corresponding time value" is one
	// that should never be uttered in kind company. The `time.Unix(sec,nsec)`
	// function has a bounds for valid times that is poorly documented.
	// Said bounds are slightly outside the interval [unixUnderflow, unixOverflow]
	// for the `sec` param, ignoring nanoseconds. Things outside of this range
	// can, and possibly will be invalid due to over/underflow.
	// These bounds were found during trial and error.
	unixUnderflow = math.MinInt32 << 24
	unixOverflow  = math.MaxInt32 << 24
)

// SafeUnix is a safe way of calling a `time.Unix` without nsec added.
// This does so in a way where:
// * If the time can be safely parsed, parse it.
// * If the time can underflow, return the zero time.
// * If the time can overflow, return the max protobuf timestamp (Year 9999).
func SafeUnix(sec int64) time.Time {
	if sec < unixUnderflow {
		return time.Time{}
	} else if sec > unixOverflow {
		return time.Unix(MaxTimestampSeconds, MaxTimestampNanos)
	}
	return time.Unix(sec, 0)
}

// ToUnixMillis parses a given time into unsigned milliseconds.
// Since `UnixNano()` is used to convert, and it is not defined for years
// outside of certain bounds (see the constants for more info), we have to define
// the behaviour for this, which may limit the use of this function in other places.
//
// For a `time.Time` with a year >= 2262, we return the MaxTimestampSeconds as millis.
// For a `time.Time` with a year < 1970, we return 0
// For a `time.Time` with 1970 <= year < 2262, we return the parsed millis.
func ToUnixMillis(t time.Time) uint64 {
	// Implementation note: This isn't _exact_, as there are times within the
	// maxNanoYear that result in a valid timestamp, but this is easier/safer
	// to handle. If you're having issues with a `time.Time` in February 2262
	// that fails to parse to milliseconds properly:  ¯\_(ツ)_/¯.
	if t.Year() >= maxNanoYear {
		return MaxTimestampSeconds * 1000
	}

	// Since we are parsing to unsigned, this is the best we can do unless we
	// want to return an error.
	if t.Unix() < 0 {
		return 0
	}

	return uint64(t.UnixNano() / 1e6)
}

// FromUnixMillis parses the given uint64 as a timestamp in ms.
// We convert from uint64 to int64 within this, and as such there is a chance
// of subtle errors occurring.
//
// * If the arg underflows, return the zero time.
// * If the arg overflows, return the max timestamp time
// * Otherwise, parse it normally.
func FromUnixMillis(ms uint64) time.Time {
	// Explicitly check against uint64 -> int64 bounds
	if ms >= math.MaxInt64/1000 {
		return time.Time{}
	}

	// Check if signed time can over/underflow during a conversion
	secs := int64(ms / 1000)
	if secs < unixUnderflow {
		return time.Time{}
	}
	if secs > unixOverflow {
		return time.Unix(MaxTimestampSeconds, MaxTimestampNanos)
	}

	return time.Unix(secs, int64(ms%1000)*int64(time.Millisecond))
}

// boundTime bounds a `time.Time` to be within the valid timestamp range.
// This allows it to be used in a timestamp marshal without risk of error.
// * Anything above the range gets rounded to the max timestamp.
// * Anything below the range gets rounded to the zero time.
// * Otherwise just returns the given time.
func boundTime(t time.Time) time.Time {
	u := t.Unix()
	if t.Year() > 9999 || u > MaxTimestampSeconds {
		return time.Unix(MaxTimestampSeconds, MaxTimestampNanos).In(t.Location())
	}
	if t.Year() < 0 || u < MinTimestampSeconds {
		return time.Time{}.In(t.Location())
	}

	return t
}

// ToTimestamp parses a `time.Time` into the bounds of a timestamp, rounding
// values outside of the valid range to those inside the valid range.
func ToTimestamp(t time.Time) *timestamp.Timestamp {
	// This works because the nanos will always be bounded when calling
	// `Nanosecond` and the seconds are previously bounded by the call
	// to `BoundTime`.
	bound := boundTime(t)
	return &timestamp.Timestamp{
		Seconds: bound.Unix(),
		Nanos:   int32(bound.Nanosecond()),
	}
}

// FromTimestamp parses a `ptypes.Timestamp` into the bounds of a timestamp.
//
// * A nil timestamp will be parsed to a zero `time.Time`.
// * A timestamp outside of the nano range will be corrected, then re-parsed.
// * A timestamp outside of either second bound will be parsed to the correct limit.
func FromTimestamp(ts *timestamp.Timestamp) time.Time {
	sec := ts.GetSeconds()
	nsec := int64(ts.GetNanos())

	// Fix the nano seconds if we can, taken from the `time.Unix(sec,nsec)`
	// function in the standard library.
	if nsec < 0 || nsec >= 1e9 {
		n := nsec / 1e9
		sec += n
		nsec -= n * 1e9
		if nsec < 0 {
			nsec += 1e9
			sec--
		}
	}

	// Now parse accordingly.
	if sec < MinTimestampSeconds {
		return time.Time{}
	} else if sec > MaxTimestampSeconds {
		return time.Unix(MaxTimestampSeconds, MaxTimestampNanos)
	}

	return time.Unix(sec, nsec)
}

// MaxTimestamp returns the maximum timestamp possible, which corresponds to
// having a seconds field of `253402300799`, and a nanoseconds field of 1e9-1.
func MaxTimestamp() *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: MaxTimestampSeconds,
		Nanos:   int32(MaxTimestampNanos),
	}
}
