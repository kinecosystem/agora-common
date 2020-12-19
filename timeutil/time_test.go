package timeutil

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func TestSafeUnix(t *testing.T) {
	tests := []struct {
		name string
		sec  int64
		want time.Time
	}{
		{
			name: "Zero",
			sec:  0,
			want: time.Unix(0, 0),
		},
		{
			name: "Now",
			sec:  time.Now().Round(time.Second).Unix(),
			want: time.Now().Round(time.Second),
		},
		{
			name: "MaxInt32",
			sec:  math.MaxInt32,
			want: time.Unix(math.MaxInt32, 0),
		},
		{
			name: "MinInt32",
			sec:  math.MinInt32,
			want: time.Unix(math.MinInt32, 0),
		},
		{
			name: "MaxInt64",
			sec:  math.MaxInt64,
			want: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name: "MinInt64",
			sec:  math.MinInt64,
			want: time.Time{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Seconds: %d, Expected: %s", tt.sec, tt.want.String())
			if got := SafeUnix(tt.sec); !tt.want.Equal(got) {
				t.Errorf("SafeUnix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUnixMillis(t *testing.T) {
	tests := []struct {
		name string
		arg  time.Time
		want uint64
	}{
		{
			name: "Now",
			arg:  time.Now(),
			want: uint64(time.Now().Unix()) * 1000,
		},
		{
			name: "Zero",
			arg:  time.Time{},
			want: 0,
		},
		{
			name: "Pre-Zero",
			arg:  time.Time{}.Add(-5 * time.Hour),
			want: 0,
		},
		{
			name: "Post-Zero",
			arg:  time.Time{}.Add(1 * time.Millisecond),
			want: 0,
		},
		{
			name: "Unix Zero",
			arg:  time.Unix(0, 0),
			want: 0,
		},
		{
			name: "Pre-Unix Zero",
			arg:  time.Unix(-1, 0),
			want: 0,
		},
		{
			name: "Post-Unix Zero",
			arg:  time.Unix(1, 0),
			want: 1000,
		},
		{
			name: "Outside Date Range",
			arg:  time.Date(2262, 1, 1, 0, 0, 0, 0, time.UTC),
			want: MaxTimestampSeconds * 1000,
		},
		{
			name: "Inside Date Range",
			arg:  time.Date(2262, 1, 1, 0, 0, 0, 0, time.UTC).Add(-1 * time.Millisecond),
			want: uint64(time.Date(2261, 12, 31, 23, 59, 59, 999, time.UTC).Unix()) * 1000,
		},
		{
			name: "Max Timestamp Seconds",
			arg:  time.Unix(MaxTimestampSeconds, 0),
			want: MaxTimestampSeconds * 1000,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Function called with: %s, Expected result: %d", tt.arg.String(), tt.want)
			if got := ToUnixMillis(tt.arg); (got/1000)*1000 != tt.want {
				t.Errorf("ToUnixMillis() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromUnixMillis(t *testing.T) {
	tests := []struct {
		name string
		arg  uint64
		want time.Time
	}{
		{
			name: "Unix Zero",
			arg:  0,
			want: time.Unix(0, 0),
		},
		{
			name: "MaxInt64 Overflow",
			arg:  math.MaxInt64 + 1,
			want: time.Time{},
		},
		{
			name: "MaxInt32",
			arg:  math.MaxInt32,
			want: time.Unix(math.MaxInt32/1000, int64(math.MaxInt32%1000*time.Millisecond)),
		},
		{
			name: "Max Timestamp Seconds",
			arg:  MaxTimestampSeconds * 1000,
			want: time.Unix(MaxTimestampSeconds, 0),
		},
		{
			name: "Second after Epoch",
			arg:  1,
			want: time.Unix(0, 0).Add(1 * time.Millisecond),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := FromUnixMillis(tt.arg); !tt.want.Equal(got) {
				t.Errorf("FromUnixMillis() = %v, want %v", got.UTC(), tt.want.UTC())
			}
		})
	}
}

func TestBoundTime(t *testing.T) {
	tests := []struct {
		name  string
		given time.Time
		want  time.Time
	}{
		{
			name:  "Before Zero",
			given: time.Unix(MinTimestampSeconds-1, 0),
			want:  time.Time{},
		},
		{
			name:  "Nanosecond Before Zero",
			given: time.Unix(MinTimestampSeconds, -1),
			want:  time.Time{},
		},
		{
			name:  "Zero",
			given: time.Time{},
			want:  time.Time{},
		},
		{
			name:  "After Zero",
			given: time.Unix(MinTimestampSeconds+8, 0),
			want:  time.Time{}.Add(8 * time.Second),
		},
		{
			name:  "Now",
			given: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
			want:  time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name:  "Before Max",
			given: time.Unix(MaxTimestampSeconds-8, 0),
			want:  time.Unix(MaxTimestampSeconds-8, 0),
		},
		{
			name:  "After Max Seconds",
			given: time.Unix(MaxTimestampSeconds, 1),
			want:  time.Unix(MaxTimestampSeconds, 1),
		},
		{
			name:  "Max",
			given: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
			want:  time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name:  "Nanosecond After Max",
			given: time.Unix(MaxTimestampSeconds, 1e9),
			want:  time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name:  "After Max",
			given: time.Unix(MaxTimestampSeconds+8, 0),
			want:  time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := boundTime(tt.given); !tt.want.Equal(got) {
				t.Errorf("boundTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToTimestamp(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		given time.Time
		want  *timestamp.Timestamp
	}{
		{
			name:  "Before Zero",
			given: time.Time{}.Add(-5 * time.Second),
			want:  mustTimestamp(t, time.Time{}),
		},
		{
			name:  "Nanosecond Before Zero",
			given: time.Time{}.Add(-1 * time.Nanosecond),
			want:  mustTimestamp(t, time.Time{}),
		},
		{
			name:  "Zero",
			given: time.Time{},
			want:  mustTimestamp(t, time.Time{}),
		},
		{
			name:  "After Zero",
			given: time.Time{}.Add(5 * time.Minute),
			want:  mustTimestamp(t, time.Time{}.Add(5*time.Minute)),
		},
		{
			name:  "Now",
			given: now,
			want:  mustTimestamp(t, now),
		},
		{
			name:  "Before Max",
			given: time.Unix(MaxTimestampSeconds-1, 0),
			want:  mustTimestamp(t, time.Unix(MaxTimestampSeconds-1, 0)),
		},
		{
			name:  "Max",
			given: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
			want:  mustTimestamp(t, time.Unix(MaxTimestampSeconds, MaxTimestampNanos)),
		},
		{
			name:  "Nanosecond After Max",
			given: time.Unix(MaxTimestampSeconds, MaxTimestampNanos+1),
			want:  mustTimestamp(t, time.Unix(MaxTimestampSeconds, MaxTimestampNanos)),
		},
		{
			name:  "After Max",
			given: time.Unix(MaxTimestampSeconds+1, 0),
			want:  mustTimestamp(t, time.Unix(MaxTimestampSeconds, MaxTimestampNanos)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToTimestamp(tt.given); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromTimestamp(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name  string
		given *timestamp.Timestamp
		want  time.Time
	}{
		{
			name: "Before Zero",
			given: &timestamp.Timestamp{
				Seconds: MinTimestampSeconds - 1,
				Nanos:   0,
			},
			want: time.Unix(MinTimestampSeconds, 0),
		},
		{
			name: "Nanosecond Before Zero",
			given: &timestamp.Timestamp{
				Seconds: MinTimestampSeconds,
				Nanos:   -1,
			},
			want: time.Unix(MinTimestampSeconds, 0),
		},
		{
			name:  "Zero",
			given: mustTimestamp(t, time.Time{}),
			want:  time.Unix(MinTimestampSeconds, 0),
		},
		{
			name:  "After Zero",
			given: mustTimestamp(t, time.Time{}.Add(5*time.Minute)),
			want:  time.Time{}.Add(5 * time.Minute),
		},
		{
			name:  "Now",
			given: mustTimestamp(t, now),
			want:  now,
		},
		{
			name:  "Before Max",
			given: mustTimestamp(t, time.Unix(MaxTimestampSeconds-1, 0)),
			want:  time.Unix(MaxTimestampSeconds-1, 0),
		},
		{
			name:  "After Max Seconds",
			given: mustTimestamp(t, time.Unix(MaxTimestampSeconds, 1)),
			want:  time.Unix(MaxTimestampSeconds, 1),
		},
		{
			name:  "Max",
			given: mustTimestamp(t, time.Unix(MaxTimestampSeconds, MaxTimestampNanos)),
			want:  time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name: "Nanosecond After Max",
			given: &timestamp.Timestamp{
				Seconds: MaxTimestampSeconds,
				Nanos:   1e9,
			},
			want: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
		{
			name: "After Max",
			given: &timestamp.Timestamp{
				Seconds: MaxTimestampSeconds + 1,
				Nanos:   0,
			},
			want: time.Unix(MaxTimestampSeconds, MaxTimestampNanos),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromTimestamp(tt.given); !tt.want.Equal(got) {
				t.Errorf("FromTimestamp() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustTimestamp(t *testing.T, date time.Time) *timestamp.Timestamp {
	ts, err := ptypes.TimestampProto(date)
	if err != nil {
		t.Errorf("Failed to parse timestamp, given err: %s", err.Error())
	}
	return ts
}
