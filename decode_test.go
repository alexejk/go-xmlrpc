package xmlrpc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdDecoder_DecodeRaw(t *testing.T) {
	tests := map[string]struct {
		testFile    string
		skipUnknown bool
		v           interface{}
		expect      interface{}
		err         error
	}{
		"simple response": {
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
		"array response": {
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
		"array response - mixed content": {
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
		"array response - bad param": {
			testFile: "response_array.xml",
			v: &struct {
				Ints string // <- This is unexpected type
			}{},
			expect: nil,
			err:    fmt.Errorf(errFormatInvalidFieldType, "slice", "string"),
		},
		"struct response": {
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
		"struct response - skip unknown": {
			testFile:    "response_struct.xml",
			skipUnknown: true,
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
		"struct response - bad param": {
			testFile: "response_struct.xml",
			v: &struct {
				Struct string // <- This is unexpected type
			}{},
			expect: nil,
			err:    fmt.Errorf(errFormatInvalidFieldTypeOrType, "struct", "map", "string"),
		},
		"struct response empty values (explicit)": {
			testFile: "response_struct_empty_values.xml",
			v: &struct {
				Struct struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}
			}{},
			expect: &struct {
				Struct struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}
			}{
				Struct: struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}{
					EmptyString: ``,
					EmptyInt:    0,
					EmptyInt4:   0,
					EmptyDouble: 0,
					EmptyDate:   time.Time{},
					EmptyBase64: nil,
					EmptyArray:  nil,
				},
			},
		},
		"struct response empty values (implicit)": {
			testFile: "response_struct_empty_values.xml",
			v: &struct {
				Struct struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}
			}{},
			expect: &struct {
				Struct struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}
			}{
				Struct: struct {
					EmptyString  string
					EmptyInt     int
					EmptyInt4    int
					EmptyDouble  int
					EmptyBoolean bool
					EmptyDate    time.Time
					EmptyBase64  []byte
					EmptyArray   []any
				}{},
			},
		},
	}

	for tName, tt := range tests {
		t.Run(tName, func(t *testing.T) {
			dec := &StdDecoder{}
			dec.skipUnknownFields = tt.skipUnknown
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

func TestStdDecoder_DecodeRaw_Arrays(t *testing.T) {
	type TestStruct struct {
		Array []any
	}

	tests := map[string]struct {
		testFile string
		expect   *TestStruct
		err      error
	}{
		"Basic mixed array": {
			testFile: "response_array_mixed.xml",
			expect: &TestStruct{
				Array: []any{10, "s11", true},
			},
		},
		"Basic mixed array - missing type declarations": {
			testFile: "response_array_mixed_missing_types.xml",
			expect: &TestStruct{
				Array: []any{0, "4099", "O3D217AC", "<c><b>123</b></c>"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dec := &StdDecoder{}
			decodeTarget := &TestStruct{}
			err := dec.DecodeRaw(loadTestFile(t, tt.testFile), decodeTarget)
			if tt.err == nil {
				require.NoError(t, err)
				require.EqualValues(t, tt.expect, decodeTarget)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.err, err)
			}
		})
	}
}

func TestStdDecoder_DecodeRaw_Struct_Map(t *testing.T) {
	type DataType struct {
		Id      string `json:"id" xmlrpc:"id"`
		PubDate string `json:"pub_date" xmlrpc:"pub_date"`
		Title   string `json:"title" xmlrpc:"title"`
	}

	type TestResponse struct {
		Data map[string][][]DataType
	}

	tests := map[string]struct {
		testFile string
		v        interface{}
		expect   interface{}
		err      error
	}{
		"Basic struct to map": {
			testFile: "response_struct.xml",
			v: &struct {
				Data map[string]any
			}{},
			expect: &struct {
				Data map[string]any
			}{
				Data: map[string]any{
					"foo":          "bar",
					"baz":          2,
					"woBleBobble":  true,
					"WoBleBobble2": 34,
					"2":            3,
				},
			},
		},
		"Invalid key type": {
			testFile: "response_struct.xml",
			v: &struct {
				Data map[any]any
			}{},
			err: fmt.Errorf(errFormatInvalidMapKeyTypeForStruct, "interface"),
		},
		"Nested structs to map": {
			testFile: "response_nested_random_struct.xml",
			v:        &TestResponse{},
			expect: &TestResponse{
				Data: map[string][][]DataType{
					"TESTING1": {
						{
							{Id: "1009470", PubDate: "2020-01-11 00:00:00", Title: "TITLE"},
							{Id: "1009879", PubDate: "2020-01-11 00:00:00", Title: "TITLE2"},
							{Id: "1304451", PubDate: "2020-01-13 17:16:49", Title: "Title3"},
						},
					},
					"TESTING2": {
						{
							{Id: "1329812", PubDate: "2020-01-11 00:00:00", Title: "NewTitle"},
							{Id: "1489372", PubDate: "2021-01-11 00:00:00", Title: "NextTitle"},
							{Id: "1229276", PubDate: "2020-01-13 17:16:49", Title: "Title12"},
						},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dec := &StdDecoder{}
			decodeTarget := tt.v
			err := dec.DecodeRaw(loadTestFile(t, tt.testFile), decodeTarget)
			if tt.err == nil {
				require.NoError(t, err)
				require.EqualValues(t, tt.expect, decodeTarget)
			} else {
				require.Error(t, err)
				require.Equal(t, tt.err, err)
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

// Issue: https://github.com/alexejk/go-xmlrpc/issues/84
func Test_github_84(t *testing.T) {
	dec := &StdDecoder{}
	decodeTarget := struct {
		Array []any
	}{}

	err := dec.DecodeRaw(loadTestFile(t, "response_array_mixed_with_struct.xml"), &decodeTarget)
	require.NoError(t, err)
	require.Equal(t, 3, len(decodeTarget.Array))
	require.Equal(t, 200, decodeTarget.Array[0])
	require.Equal(t, "OK", decodeTarget.Array[1])
	require.Equal(t, "OK", decodeTarget.Array[2].(map[string]any)["status"])
}

func loadTestFile(t *testing.T, name string) []byte {
	path := filepath.Join("testdata", name) // relative path

	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}
