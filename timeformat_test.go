package xmlrpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()

	l, err := time.LoadLocation(name)
	require.NoError(t, err)

	return l
}

func TestLayoutTimeFormatter_FormatTime(t *testing.T) {
	stockholm := mustLoadLocation(t, "Europe/Stockholm")

	tests := []struct {
		name      string
		formatter *LayoutTimeFormatter
		input     time.Time
		expect    string
	}{
		{
			name:      "zero-value formatter defaults to RFC3339 - UTC",
			formatter: &LayoutTimeFormatter{},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC),
			expect:    "2019-10-11T13:40:30Z",
		},
		{
			name:      "zero-value formatter defaults to RFC3339 - offset retained",
			formatter: &LayoutTimeFormatter{},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, stockholm),
			expect:    "2019-10-11T13:40:30+02:00",
		},
		{
			name:      "compact layout",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC),
			expect:    "20191011T13:40:30",
		},
		{
			name:      "compact layout keeps wall-clock of value location by default",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, stockholm),
			expect:    "20191011T13:40:30",
		},
		{
			name: "FormatLocation converts before formatting",
			formatter: &LayoutTimeFormatter{
				FormatLayout:   LayoutISO8601Compact,
				FormatLocation: time.UTC,
			},
			input:  time.Date(2019, 10, 11, 13, 40, 30, 0, stockholm),
			expect: "20191011T11:40:30",
		},
		{
			name: "FormatLocation to non-UTC",
			formatter: &LayoutTimeFormatter{
				FormatLayout:   LayoutISO8601Compact,
				FormatLocation: stockholm,
			},
			input:  time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC),
			expect: "20191011T15:40:30",
		},
		{
			name:      "compact zoned layout",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601CompactZoned},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, stockholm),
			expect:    "20191011T13:40:30+0200",
		},
		{
			name:      "ParseLayouts does not affect encoding",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, ParseLayouts: CommonParseLayouts()},
			input:     time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC),
			expect:    "20191011T13:40:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expect, tt.formatter.FormatTime(tt.input))
		})
	}
}

func TestLayoutTimeFormatter_ParseTime(t *testing.T) {
	stockholm := mustLoadLocation(t, "Europe/Stockholm")

	tests := []struct {
		name      string
		formatter *LayoutTimeFormatter
		input     string
		// expect is the parsed value rendered as RFC3339, capturing both instant and offset
		expect    string
		expectErr bool
	}{
		{
			name:      "zero-value formatter accepts RFC3339",
			formatter: &LayoutTimeFormatter{},
			input:     "2019-10-11T13:40:30Z",
			expect:    "2019-10-11T13:40:30Z",
		},
		{
			name:      "zero-value formatter retains offset",
			formatter: &LayoutTimeFormatter{},
			input:     "2019-10-11T13:40:30+02:00",
			expect:    "2019-10-11T13:40:30+02:00",
		},
		{
			name:      "zero-value formatter rejects compact form",
			formatter: &LayoutTimeFormatter{},
			input:     "20191011T13:40:30",
			expectErr: true,
		},
		{
			name:      "compact layout defaults to UTC",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact},
			input:     "20191011T13:40:30",
			expect:    "2019-10-11T13:40:30Z",
		},
		{
			name: "ParseLocation applied to zone-less value",
			formatter: &LayoutTimeFormatter{
				FormatLayout:  LayoutISO8601Compact,
				ParseLocation: stockholm,
			},
			input:  "20191011T13:40:30",
			expect: "2019-10-11T13:40:30+02:00",
		},
		{
			name: "ParseLocation ignored when value carries an offset",
			formatter: &LayoutTimeFormatter{
				FormatLayout:  LayoutISO8601Compact,
				ParseLayouts:  CommonParseLayouts(),
				ParseLocation: stockholm,
			},
			input:  "20191011T13:40:30-0700",
			expect: "2019-10-11T13:40:30-07:00",
		},
		{
			name:      "ParseLayouts accepts extended form",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, ParseLayouts: CommonParseLayouts()},
			input:     "2019-10-11T13:40:30Z",
			expect:    "2019-10-11T13:40:30Z",
		},
		{
			name:      "ParseLayouts is the complete list - FormatLayout is not appended",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, ParseLayouts: []string{LayoutISO8601Extended}},
			input:     "20191011T13:40:30",
			expectErr: true,
		},
		{
			name:      "no matching layout",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, ParseLayouts: CommonParseLayouts()},
			input:     "11/10/2019 13:40",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.formatter.ParseTime(tt.input)
			if tt.expectErr {
				require.Error(t, err)
				require.True(t, got.IsZero(), "failed parse must return zero time")

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expect, got.Format(time.RFC3339))
		})
	}
}

func TestLayoutTimeFormatter_ParseTime_UnsetLocationMatchesStdlib(t *testing.T) {
	// An offset matching the process timezone is the only case where time.Parse and
	// time.ParseInLocation differ: Parse keeps the real location and its DST rules.
	// time.Local is pinned so this holds regardless of where the tests run.
	original := time.Local
	t.Cleanup(func() { time.Local = original })
	time.Local = time.FixedZone("TEST", 2*60*60)

	ref := time.Date(2019, 7, 17, 14, 8, 55, 0, time.Local)
	value := ref.Format(time.RFC3339)
	require.Equal(t, "2019-07-17T14:08:55+02:00", value)

	want, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)

	got, err := (&LayoutTimeFormatter{}).ParseTime(value)
	require.NoError(t, err)

	require.Equal(t, want, got, "unset ParseLocation must behave exactly as time.Parse")
	require.Equal(t, want.Location().String(), got.Location().String())

	// An explicit ParseLocation keeps an offset carried by the value ...
	f := &LayoutTimeFormatter{ParseLayouts: []string{time.RFC3339, LayoutISO8601Extended}, ParseLocation: time.UTC}

	kept, err := f.ParseTime(value)
	require.NoError(t, err)
	require.True(t, want.Equal(kept), "instant must be preserved either way")
	require.Equal(t, "2019-07-17T14:08:55+02:00", kept.Format(time.RFC3339))

	// ... and applies only to values that carry none, regardless of time.Local
	applied, err := f.ParseTime("2019-07-17T14:08:55")
	require.NoError(t, err)
	require.Equal(t, "2019-07-17T14:08:55Z", applied.Format(time.RFC3339))
}

func TestLayoutTimeFormatter_RoundTrip(t *testing.T) {
	layouts := []string{
		LayoutISO8601Compact,
		LayoutISO8601CompactZoned,
		LayoutISO8601Basic,
		LayoutISO8601BasicZoned,
		LayoutISO8601Extended,
		LayoutISO8601ExtendedZoned,
	}

	want := time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC)

	for _, layout := range layouts {
		t.Run(layout, func(t *testing.T) {
			f := &LayoutTimeFormatter{FormatLayout: layout}

			got, err := f.ParseTime(f.FormatTime(want))
			require.NoError(t, err)
			require.True(t, want.Equal(got), "expected %s, got %s", want, got)
		})
	}
}

func TestLayoutTimeFormatter_RoundTrip_NonUTC(t *testing.T) {
	want := time.Date(2019, 10, 11, 13, 40, 30, 0, mustLoadLocation(t, "Europe/Stockholm"))

	t.Run("zoned layouts preserve the instant", func(t *testing.T) {
		for _, layout := range []string{LayoutISO8601CompactZoned, LayoutISO8601BasicZoned, LayoutISO8601ExtendedZoned} {
			f := &LayoutTimeFormatter{FormatLayout: layout}

			got, err := f.ParseTime(f.FormatTime(want))
			require.NoError(t, err)
			require.True(t, want.Equal(got), "layout %s: expected %s, got %s", layout, want, got)
		}
	})

	// Zone-less layouts drop the offset, so the wall-clock survives and the instant does
	// not. FormatLocation is what makes these layouts safe.
	t.Run("zone-less layouts shift the instant without FormatLocation", func(t *testing.T) {
		f := &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact}

		got, err := f.ParseTime(f.FormatTime(want))
		require.NoError(t, err)
		require.False(t, want.Equal(got), "instant must not survive a zone-less round trip")
		require.Equal(t, "2019-10-11T13:40:30Z", got.Format(time.RFC3339))
	})

	t.Run("zone-less layouts preserve the instant with FormatLocation", func(t *testing.T) {
		f := &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, FormatLocation: time.UTC}

		got, err := f.ParseTime(f.FormatTime(want))
		require.NoError(t, err)
		require.True(t, want.Equal(got), "expected %s, got %s", want, got)
	})
}

func TestLayoutTimeFormatter_ParseTime_WrapsParseError(t *testing.T) {
	_, err := (&LayoutTimeFormatter{}).ParseTime("not-a-time")

	var parseErr *time.ParseError
	require.ErrorAs(t, err, &parseErr, "underlying *time.ParseError must stay unwrappable")
}

func Test_parseLayouts(t *testing.T) {
	tests := []struct {
		name      string
		formatter *LayoutTimeFormatter
		expect    []string
	}{
		{
			name:      "zero-value falls back to RFC3339",
			formatter: &LayoutTimeFormatter{},
			expect:    []string{LayoutISO8601ExtendedZoned},
		},
		{
			name:      "format layout only",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact},
			expect:    []string{LayoutISO8601Compact},
		},
		{
			name:      "parse layouts used verbatim",
			formatter: &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact, ParseLayouts: []string{LayoutISO8601Extended}},
			expect:    []string{LayoutISO8601Extended},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expect, tt.formatter.parseLayouts())
		})
	}
}

func TestCommonParseLayouts_AcceptedForms(t *testing.T) {
	f := &LayoutTimeFormatter{ParseLayouts: CommonParseLayouts()}

	tests := []struct {
		input string
		// expect is the parsed value as RFC3339; empty means the input must be rejected
		expect string
	}{
		{input: "19980717T14:08:55", expect: "1998-07-17T14:08:55Z"},
		{input: "19980717T14:08:55Z", expect: "1998-07-17T14:08:55Z"},
		{input: "19980717T14:08:55+0200", expect: "1998-07-17T14:08:55+02:00"},
		{input: "19980717T140855", expect: "1998-07-17T14:08:55Z"},
		{input: "19980717T140855Z", expect: "1998-07-17T14:08:55Z"},
		{input: "19980717T140855-0700", expect: "1998-07-17T14:08:55-07:00"},
		{input: "1998-07-17T14:08:55", expect: "1998-07-17T14:08:55Z"},
		{input: "1998-07-17T14:08:55Z", expect: "1998-07-17T14:08:55Z"},
		{input: "1998-07-17T14:08:55+02:00", expect: "1998-07-17T14:08:55+02:00"},
		// A fractional second is accepted even though no layout declares one
		{input: "1998-07-17T14:08:55.123Z", expect: "1998-07-17T14:08:55Z"},
		{input: "19980717T140855.123", expect: "1998-07-17T14:08:55Z"},
		// Date-only and time-only values must not match a longer layout
		{input: "19980717", expect: ""},
		{input: "140855", expect: ""},
		{input: "1998-07-17 14:08:55", expect: ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := f.ParseTime(tt.input)
			if tt.expect == "" {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expect, got.Format(time.RFC3339))
		})
	}
}

func TestLayoutTimeFormatter_ParseTime_FractionalSeconds(t *testing.T) {
	f := &LayoutTimeFormatter{ParseLayouts: CommonParseLayouts()}

	for _, input := range []string{"1998-07-17T14:08:55.123Z", "19980717T140855.123", "19980717T14:08:55.123"} {
		t.Run(input, func(t *testing.T) {
			got, err := f.ParseTime(input)
			require.NoError(t, err)
			require.Equal(t, 123000000, got.Nanosecond(), "fractional seconds must be retained")
		})
	}
}

func TestLayoutTimeFormatter_NilReceiver(t *testing.T) {
	var f *LayoutTimeFormatter

	ts := time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC)
	require.Equal(t, "2019-10-11T13:40:30Z", f.FormatTime(ts))

	got, err := f.ParseTime("2019-10-11T13:40:30Z")
	require.NoError(t, err)
	require.True(t, ts.Equal(got))
}

func TestCommonParseLayouts(t *testing.T) {
	first := CommonParseLayouts()
	require.NotEmpty(t, first)

	first[0] = "mutated"
	require.NotEqual(t, "mutated", CommonParseLayouts()[0], "must return a fresh slice")
}

func Test_timeFormatterOrDefault(t *testing.T) {
	require.Same(t, defaultTimeFormatter, timeFormatterOrDefault(nil))

	custom := &LayoutTimeFormatter{FormatLayout: LayoutISO8601Compact}
	require.Same(t, custom, timeFormatterOrDefault(custom))
}
