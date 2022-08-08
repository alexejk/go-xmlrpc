package xmlrpc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStdDecoder_DecodeRaw(t *testing.T) {
	tests := []struct {
		name     string
		testFile string
		v        interface{}
		expect   interface{}
		err      error
	}{
		{
			name:     "simple response",
			testFile: "response_simple.xml",
			v: &struct {
				Param string
				Int   int
			}{},
			expect: &struct {
				Param string
				Int   int
			}{
				Param: "South Dakota",
				Int:   12345,
			},
		},
		{
			name:     "array response",
			testFile: "response_array.xml",
			v: &struct {
				Ints []int
			}{},
			expect: &struct {
				Ints []int
			}{
				Ints: []int{
					10, 11, 12,
				},
			},
		},
		{
			name:     "array response - mixed content",
			testFile: "response_array_mixed.xml",
			v: &struct {
				Mixed []interface{}
			}{},
			expect: &struct {
				Mixed []interface{}
			}{
				Mixed: []interface{}{
					10, "s11", true,
				},
			},
		},
		{
			name:     "array response - bad param",
			testFile: "response_array.xml",
			v: &struct {
				Ints string // <- This is unexpected type
			}{},
			expect: nil,
			err:    fmt.Errorf(errFormatInvalidFieldType, "slice", "string"),
		},
		{
			name:     "struct response",
			testFile: "response_struct.xml",
			v: &struct {
				Struct struct {
					Foo          string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}
			}{},
			expect: &struct {
				Struct struct {
					Foo          string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}
			}{
				Struct: struct {
					Foo          string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}{
					Foo:          "bar",
					Baz:          2,
					WoBleBobble:  true,
					WoBleBobble2: 34,
					Field2:       3,
				},
			},
		},
		{
			name:     "struct response - bad param",
			testFile: "response_struct.xml",
			v: &struct {
				Struct string // <- This is unexpected type
			}{},
			expect: nil,
			err:    fmt.Errorf(errFormatInvalidFieldType, "struct", "string"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := &StdDecoder{}
			err := dec.DecodeRaw(loadTestFile(t, tt.testFile), tt.v)

			if tt.err == nil {
				require.NoError(t, err)
				require.EqualValues(t, tt.expect, tt.v)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func TestStdDecoder_DecodeRaw_StructFields(t *testing.T) {
	type StrAlias string
	type IntAlias int

	sPtr := func(v string) *string {
		return &v
	}
	iPtr := func(v int) *int {
		return &v
	}

	tests := []struct {
		name     string
		testFile string
		v        interface{}
		expect   interface{}
		err      error
	}{
		{
			name:     "struct convertible string alias",
			testFile: "response_struct.xml",
			v: &struct {
				Struct struct {
					Foo          StrAlias
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}
			}{},
			expect: &struct {
				Struct struct {
					Foo          StrAlias
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}
			}{
				Struct: struct {
					Foo          StrAlias
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
					Field2       int `xmlrpc:"2"`
				}{
					Foo:          "bar",
					Baz:          2,
					WoBleBobble:  true,
					WoBleBobble2: 34,
					Field2:       3,
				},
			},
		},
		{
			name:     "struct pointer",
			testFile: "response_struct.xml",
			v: &struct {
				Struct *struct {
					Foo          *string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 *int
					Field2       *int `xmlrpc:"2"`
				}
			}{},
			expect: &struct {
				Struct *struct {
					Foo          *string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 *int
					Field2       *int `xmlrpc:"2"`
				}
			}{
				Struct: &struct {
					Foo          *string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 *int
					Field2       *int `xmlrpc:"2"`
				}{
					Foo:          sPtr("bar"),
					Baz:          2,
					WoBleBobble:  true,
					WoBleBobble2: iPtr(34),
					Field2:       iPtr(3),
				},
			},
		},
		{
			name:     "struct non-convertible type",
			testFile: "response_struct.xml",
			v: &struct {
				Struct struct {
					Foo          IntAlias
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
				}
			}{},
			err: errors.New("type 'xmlrpc.IntAlias' cannot be assigned a value of type 'string'"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dec := &StdDecoder{}
			err := dec.DecodeRaw(loadTestFile(t, tt.testFile), tt.v)
			if tt.err == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.err, errors.Unwrap(err))
			}

			if tt.err == nil {
				require.EqualValues(t, tt.expect, tt.v)
			}
		})
	}
}

func TestStdDecoder_DecodeRaw_Fault(t *testing.T) {
	decodeTarget := &struct {
		Ints []int
	}{}
	dec := &StdDecoder{}
	err := dec.DecodeRaw(loadTestFile(t, "response_fault.xml"), decodeTarget)
	require.Error(t, err)

	fT := &Fault{}
	require.True(t, errors.As(err, &fT))
	require.EqualValues(t, &Fault{
		Code:   4,
		String: "Too many parameters.",
	}, fT)
}

func Test_fieldsMustEqual(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect int
		err    error
	}{
		{
			name:   "empty struct",
			input:  struct{}{},
			expect: 0,
		},
		{
			name: "no exported fields",
			input: struct {
				priv int
			}{
				priv: 3,
			},
			expect: 0,
		},
		{
			name: "exported fields",
			input: struct {
				Pub int
			}{
				Pub: 3,
			},
			expect: 1,
		},
		{
			name: "mixed exported/unexported fields",
			input: struct {
				priv int
				Pub  int
			}{
				Pub:  3,
				priv: 4,
			},
			expect: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fieldsMustEqual(tt.input, tt.expect)
			require.Equal(t, tt.err, err)
		})
	}
}

func Test_findFieldByNameOrTag(t *testing.T) {
	v := &struct {
		Normal       string
		Renamed      string `xmlrpc:"222"`
		SkipMe       string `xmlrpc:"-"`
		UseMeInstead string `xmlrpc:"SkipMe,unknown-opt"`
	}{
		Normal:       "xxx",
		Renamed:      "yyy",
		SkipMe:       "skipMe",
		UseMeInstead: "Don't Skip Me",
	}

	// Normal fetch
	normField := reflect.Indirect(reflect.ValueOf(v)).FieldByName("Normal")
	normFieldFound := findFieldByNameOrTag(reflect.Indirect(reflect.ValueOf(v)), "Normal")
	require.Equal(t, normField, normFieldFound)
	require.True(t, normFieldFound.IsValid())

	// Basic remapping
	renamedField := reflect.Indirect(reflect.ValueOf(v)).FieldByName("Renamed")
	renamedFieldFound := findFieldByNameOrTag(reflect.Indirect(reflect.ValueOf(v)), "222")
	require.Equal(t, renamedField, renamedFieldFound)
	require.True(t, renamedFieldFound.IsValid())

	// Remapping with skip
	// Actual field "SkipMe" is ignored, and struct field "SkipMe" is remapped to a "UseMeInstead"
	skipField := reflect.Indirect(reflect.ValueOf(v)).FieldByName("UseMeInstead")
	skipFieldFound := findFieldByNameOrTag(reflect.Indirect(reflect.ValueOf(v)), "SkipMe")
	require.Equal(t, skipField, skipFieldFound)
	require.True(t, skipFieldFound.IsValid())
	require.Equal(t, "Don't Skip Me", skipFieldFound.String())
}

func Test_structMemberToFieldName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "lower-camel-case",
			input:  "myField",
			expect: "MyField",
		},
		{
			name:   "lower-snake-case",
			input:  "my_field",
			expect: "MyField",
		},
		{
			name:   "upper-snake-case",
			input:  "my_Field",
			expect: "MyField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := structMemberToFieldName(tt.input)
			require.Equal(t, tt.expect, r)
		})
	}
}

func loadTestFile(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}
