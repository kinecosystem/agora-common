package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsISO8601(t *testing.T) {
	assert.True(t, IsISO8601("-P2dT+4H45M+10.123456789s"))
	assert.False(t, IsISO8601("34.5s"))
}

func TestParseISO8601(t *testing.T) {
	durationStr := "-P2dT+4H45M+10.123456789s"
	durationExpected := -(48*time.Hour + 4*time.Hour + 45*time.Minute + 10*time.Second + 123456789*time.Nanosecond)
	val, err := ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no time section
	durationStr = "P106751D"
	durationExpected = 106751 * 24 * time.Hour
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no days
	durationStr = "PT-4H45M10.123456789S"
	durationExpected = -4*time.Hour + 45*time.Minute + 10*time.Second + 123456789*time.Nanosecond
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no hours
	durationStr = "PT45M10.123456789S"
	durationExpected = 45*time.Minute + 10*time.Second + 123456789*time.Nanosecond
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no minutes
	durationStr = "pT10.123456789s"
	durationExpected = 10*time.Second + 123456789*time.Nanosecond
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no seconds
	durationStr = "PT0.123456789S"
	durationExpected = 123456789 * time.Nanosecond
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)

	// no nanos
	durationStr = "PT9223372036S"
	durationExpected = 9223372036 * time.Second
	val, err = ParseISO8601(durationStr)
	require.NoError(t, err)
	assert.Equal(t, durationExpected, val)
}

func TestParseBadFormat(t *testing.T) {
	durationStr := "PabcDT+4H45M+10.123456789S"
	_, err := ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT.2S"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "46.2s"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)
}

func TestParseOverflow(t *testing.T) {
	var durationStr string
	var err error

	// Note: 9223372036854775808 == math.MaxInt64 + 1

	// 106751 is max representable days
	durationStr = "P106751DT24H"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "P-106752D"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "P9223372036854775808D"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	// 2562047 is max representable hours
	durationStr = "PT2562047H60M"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT2562048H"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT9223372036854775808H"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	// 153722867 is max representable minutes
	durationStr = "PT-153722867M-60S"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT-153722868M"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT9223372036854775808M"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	// 9223372036 is max representable seconds
	durationStr = "PT9223372036.999999999S"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT9223372037S"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)

	durationStr = "PT9223372036854775808S"
	_, err = ParseISO8601(durationStr)
	require.Error(t, err)
}
