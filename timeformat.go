package xmlrpc

import (
	"fmt"
	"time"
)

// Layouts for the XML-RPC <dateTime.iso8601> type, for use with LayoutTimeFormatter.
//
// ISO8601 calls a form with no separators "basic" and a form with separators "extended".
// The layout shown in the XML-RPC specification's example is neither: it combines a basic
// date with an extended time, and is called compact here. The specification does not
// mandate a layout, and leaves timezone assumptions to server documentation.
const (
	// LayoutISO8601Compact is the form shown in the XML-RPC specification's example, e.g. "19980717T14:08:55".
	LayoutISO8601Compact = "20060102T15:04:05"
	// LayoutISO8601CompactZoned is the compact form with a numeric timezone offset, e.g. "19980717T14:08:55+0200".
	LayoutISO8601CompactZoned = "20060102T15:04:05Z0700"
	// LayoutISO8601Basic is the ISO8601 basic form, without any separators, e.g. "19980717T140855".
	LayoutISO8601Basic = "20060102T150405"
	// LayoutISO8601BasicZoned is the basic form with a numeric timezone offset, e.g. "19980717T140855+0200".
	LayoutISO8601BasicZoned = "20060102T150405Z0700"
	// LayoutISO8601Extended is the extended form without a timezone offset, e.g. "1998-07-17T14:08:55".
	LayoutISO8601Extended = "2006-01-02T15:04:05"
	// LayoutISO8601ExtendedZoned is the extended form with a timezone offset, e.g. "1998-07-17T14:08:55+02:00".
	// This is equivalent to time.RFC3339.
	LayoutISO8601ExtendedZoned = time.RFC3339
)

// TimeFormatter implementations convert between time.Time and the string contents of
// the XML-RPC <dateTime.iso8601> type.
//
// Implementations must be safe for concurrent use and must not be mutated once passed
// to a Client. ParseTime is never called with an empty string - empty XML values are
// decoded to the zero time.Time by the decoder.
type TimeFormatter interface {
	FormatTime(t time.Time) string
	ParseTime(value string) (time.Time, error)
}

// LayoutTimeFormatter is the default TimeFormatter implementation, driven by
// time package layout strings. A nil *LayoutTimeFormatter behaves as the zero value.
type LayoutTimeFormatter struct {
	// FormatLayout is used to encode values. Defaults to time.RFC3339.
	// Zone-less layouts should be paired with FormatLocation.
	FormatLayout string

	// FormatLocation, when set, converts values to this location before encoding.
	// This matters for zone-less layouts, where the offset is lost on the wire.
	// Defaults to nil, meaning values are encoded in whichever location they carry.
	FormatLocation *time.Location

	// ParseLayouts are attempted in order when decoding. Defaults to FormatLayout.
	// When set, it is the complete list - FormatLayout is not appended, so include it
	// to keep accepting values this client encodes. See CommonParseLayouts.
	ParseLayouts []string

	// ParseLocation resolves decoded values whose layout carries no timezone offset.
	// Values that do carry an offset always keep it.
	//
	// Defaults to nil, meaning time.Parse semantics: values without an offset are UTC,
	// and an offset matching the process timezone yields that location, keeping its DST
	// rules. Set this to decode into one location regardless of where the process runs.
	ParseLocation *time.Location
}

// CommonParseLayouts returns the dateTime.iso8601 layouts commonly encountered in
// the wild, ordered most to least specific. Intended for LayoutTimeFormatter.ParseLayouts
// when the exact form a server emits is unknown or inconsistent.
func CommonParseLayouts() []string {
	return []string{
		LayoutISO8601ExtendedZoned,
		LayoutISO8601CompactZoned,
		LayoutISO8601BasicZoned,
		LayoutISO8601Extended,
		LayoutISO8601Compact,
		LayoutISO8601Basic,
	}
}

// defaultLayoutTimeFormatter is the zero-value formatter, standing in for a nil one.
var defaultLayoutTimeFormatter = &LayoutTimeFormatter{}

// defaultTimeFormatter is used by StdEncoder and StdDecoder when no TimeFormatter is set.
var defaultTimeFormatter TimeFormatter = defaultLayoutTimeFormatter

func timeFormatterOrDefault(f TimeFormatter) TimeFormatter {
	if f == nil {
		return defaultTimeFormatter
	}

	return f
}

func (f *LayoutTimeFormatter) FormatTime(t time.Time) string {
	if f == nil {
		f = defaultLayoutTimeFormatter
	}

	if f.FormatLocation != nil {
		t = t.In(f.FormatLocation)
	}

	return t.Format(f.formatLayout())
}

func (f *LayoutTimeFormatter) ParseTime(value string) (time.Time, error) {
	if f == nil {
		f = defaultLayoutTimeFormatter
	}

	// parseLayouts never yields an empty list, so err below is always set on failure
	layouts := f.parseLayouts()

	var err error
	for _, layout := range layouts {
		var t time.Time
		if t, err = f.parse(layout, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("value '%s' does not match expected time layouts %q: %w", value, layouts, err)
}

// parse applies a single layout, honoring ParseLocation when it is set.
func (f *LayoutTimeFormatter) parse(layout, value string) (time.Time, error) {
	if f.ParseLocation == nil {
		return time.Parse(layout, value)
	}

	return time.ParseInLocation(layout, value, f.ParseLocation)
}

func (f *LayoutTimeFormatter) formatLayout() string {
	if f.FormatLayout == "" {
		return LayoutISO8601ExtendedZoned
	}

	return f.FormatLayout
}

// parseLayouts returns at least one layout - ParseTime relies on this.
func (f *LayoutTimeFormatter) parseLayouts() []string {
	if len(f.ParseLayouts) == 0 {
		return []string{f.formatLayout()}
	}

	return f.ParseLayouts
}
