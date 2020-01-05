package xmlrpc

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeResponse(t *testing.T) {
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
				}
			}{},
			expect: &struct {
				Struct struct {
					Foo          string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
				}
			}{
				Struct: struct {
					Foo          string
					Baz          int
					WoBleBobble  bool
					WoBleBobble2 int
				}{
					Foo:          "bar",
					Baz:          2,
					WoBleBobble:  true,
					WoBleBobble2: 34,
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

			err := DecodeResponse(loadTestFile(t, tt.testFile), tt.v)
			assert.Equal(t, tt.err, err)
			if tt.err == nil {
				assert.EqualValues(t, tt.expect, tt.v)
			}
		})
	}
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
			assert.Equal(t, tt.err, err)
		})
	}
}

func loadTestFile(t *testing.T, name string) string {

	path := filepath.Join("testdata", name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes)
}
