package xmlrpc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdEncoder_Encode(t *testing.T) {
	const ParamsPrefix = "<params>"
	const ParamsSuffix = "</params>"

	// methodBodyValidator is a function type that can be used to ensure the body of the method parameters is up to spec.
	type methodBodyValidator func(body string) error

	// noParamsValidator checks if the parameter body is empty, which is expected when no parameters are provided.
	noParamsValidator := func(body string) error {
		if body != "" {
			return fmt.Errorf("expected no params, got %s", body)
		}
		return nil
	}

	// exactParamsValidator checks if the parameter body matches the expected XML structure.
	// It will look for data within <params> tags.
	exactParamsValidator := func(expectation string) func(body string) error {
		return func(body string) error {
			body = strings.TrimSpace(body)
			if !strings.HasPrefix(body, ParamsPrefix) || !strings.HasSuffix(body, ParamsSuffix) {
				return fmt.Errorf("expected params body to start with <params> and end with </params>, got %s", body)
			}

			remainingBody := body[len(ParamsPrefix) : len(body)-len(ParamsSuffix)]
			if remainingBody != expectation {
				return fmt.Errorf("expected params body %s, got %s", expectation, remainingBody)
			}
			return nil
		}
	}

	// structParamWithMembersValidator checks if the parameter body contains a single struct with specific members.
	// It will look for <member> tags within the <params><param><struct> tree of tag.
	// Validation will check for the presence of only expected members and no others - if unexpected members are present, it will return an error.
	structParamWithMembersValidator := func(expectedMembers []string) func(body string) error {
		return func(body string) error {
			body = strings.TrimSpace(body)

			StructParamPrefix := fmt.Sprintf("%s<param><value><struct>", ParamsPrefix)
			StructParamSuffix := fmt.Sprintf("</struct></value></param>%s", ParamsSuffix)

			if !strings.HasPrefix(body, StructParamPrefix) || !strings.HasSuffix(body, StructParamSuffix) {
				return fmt.Errorf("expected params body to contain a single struct, got %s", body)
			}

			remainingBody := body[len(StructParamPrefix) : len(body)-len(StructParamSuffix)]

			for _, exp := range expectedMembers {
				if !strings.Contains(remainingBody, exp) {
					return fmt.Errorf("expected params body to contain %s", exp)
				}
				// removing the validated part from the remaining body
				remainingBody = strings.Replace(remainingBody, exp, "", 1)
			}

			if strings.TrimSpace(remainingBody) != "" {
				return fmt.Errorf("expected only provided members to exist, got remainder body %q", remainingBody)
			}
			return nil
		}
	}

	tests := []struct {
		name   string
		args   interface{}
		expect string
		err    string

		paramValidator methodBodyValidator
	}{
		{
			name:           "No args",
			args:           nil,
			paramValidator: noParamsValidator,
		},
		{
			name:           "Args empty struct as pointer",
			args:           &struct{}{},
			paramValidator: noParamsValidator,
		},
		{
			name:           "Args empty struct as value",
			args:           struct{}{},
			paramValidator: noParamsValidator,
		},
		{
			name: "Args as pointer",
			args: &struct {
				String string
			}{
				String: "my-name",
			},
			paramValidator: exactParamsValidator(`<param><value><string>my-name</string></value></param>`),
		},
		{
			name: "Args as value",
			args: struct {
				String string
			}{
				String: "my-name",
			},
			paramValidator: exactParamsValidator(`<param><value><string>my-name</string></value></param>`),
		},
		{
			name: "Args with unexported fields",
			args: struct {
				smthUnexported string
			}{
				smthUnexported: "i-am-unexported",
			},
			paramValidator: noParamsValidator,
		},
		{
			name: "Boolean args",
			args: &struct {
				BooleanTrue  bool
				BooleanFalse bool
			}{
				// Order purposely swapped
				BooleanFalse: false,
				BooleanTrue:  true,
			},
			paramValidator: exactParamsValidator(`<param><value><boolean>1</boolean></value></param><param><value><boolean>0</boolean></value></param>`),
		},
		{
			name: "Numerical args",
			args: &struct {
				Int    int
				Double float64
			}{
				Int:    123,
				Double: float64(12345),
			},
			paramValidator: exactParamsValidator(`<param><value><int>123</int></value></param><param><value><double>12345.000000</double></value></param>`),
		},
		{
			name: "String arg - simple",
			args: &struct {
				String string
			}{
				String: "my-name",
			},
			paramValidator: exactParamsValidator(`<param><value><string>my-name</string></value></param>`),
		},
		{
			name: "String arg - encoded",
			args: &struct {
				String string
			}{
				String: `<div class="whitespace">&nbsp;</div>`,
			},
			paramValidator: exactParamsValidator(`<param><value><string>&lt;div class=&#34;whitespace&#34;&gt;&amp;nbsp;&lt;/div&gt;</string></value></param>`),
		},
		{
			name: "Struct args - encoded",
			args: &struct {
				MyStruct struct {
					String string
				}
			}{
				MyStruct: struct {
					String string
				}{
					String: "foo",
				},
			},
			paramValidator: structParamWithMembersValidator([]string{
				`<member><name>String</name><value><string>foo</string></value></member>`,
			}),
		},
		{
			name: "Struct args renamed - encoded",
			args: &struct {
				MyStruct struct {
					String string `xmlrpc:"2-.Arg"`
				}
			}{
				MyStruct: struct {
					String string `xmlrpc:"2-.Arg"`
				}{
					String: "foo",
				},
			},
			paramValidator: structParamWithMembersValidator([]string{
				`<member><name>2-.Arg</name><value><string>foo</string></value></member>`,
			}),
		},
		{
			name: "Map-based argument of a struct",
			args: &struct {
				MyMap map[string]interface{}
			}{
				MyMap: map[string]any{
					"foo": "bar",
					"baz": 123,
				},
			},
			paramValidator: structParamWithMembersValidator([]string{
				`<member><name>foo</name><value><string>bar</string></value></member>`,
				`<member><name>baz</name><value><int>123</int></value></member>`,
			}),
		},
		{
			name: "Map-based argument without struct wrapper",
			args: map[string]any{
				"foo": "bar2",
				"baz": 123,
			},
			paramValidator: structParamWithMembersValidator([]string{
				`<member><name>foo</name><value><string>bar2</string></value></member>`,
				`<member><name>baz</name><value><int>123</int></value></member>`,
			}),
		},
		{
			name: "Map-based argument without struct wrapper - bad key type",
			args: map[int]any{
				123: "bar2",
				234: 123,
			},
			err: "unsupported type int for bare map key, only string keys are supported",
		},
		{
			name: "Unsupported argument type",
			args: 123,
			err:  "unsupported argument type int",
		},
	}

	const MethodPrefix = "<methodCall><methodName>myMethod</methodName>"
	const MethodSuffix = "</methodCall>"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.Encode(buf, "myMethod", tt.args)

			if tt.err != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.err)
				return
			}

			require.NoError(t, err)
			output := buf.String()
			require.True(t, strings.HasPrefix(output, MethodPrefix))
			require.True(t, strings.HasSuffix(output, MethodSuffix))

			body := output[len(MethodPrefix) : len(output)-len(MethodSuffix)]

			require.NoError(t, tt.paramValidator(body))
		})
	}
}

func TestStdEncoder_isByteArray(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect bool
	}{
		{
			name:   "byte array",
			input:  []byte("Something"),
			expect: true,
		},
		{
			name:   "int array",
			input:  []int{1, 2, 3},
			expect: false,
		},
		{
			name:   "uint8 array",
			input:  []uint8{1, 2, 3},
			expect: true, // byte is aliased to uint8
		},
		{
			name:   "int8 array",
			input:  []int8{1, 2, 3},
			expect: false,
		},
		{
			name:   "string",
			input:  "string here",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc := &StdEncoder{}
			resp := enc.isByteArray(tt.input)
			require.Equal(t, tt.expect, resp)
		})
	}
}

func Test_encodeArray(t *testing.T) {
	ptr := func(v string) *string {
		return &v
	}

	tests := []struct {
		name   string
		input  interface{}
		expect string
		err    error
	}{
		{
			name:   "empty slice",
			input:  []string{},
			expect: "<array><data></data></array>",
			err:    nil,
		},
		{
			name: "array of pointers",
			input: []*string{
				ptr("s1"), ptr("s2"), ptr(""), nil,
			},
			expect: "<array><data><value><string>s1</string></value><value><string>s2</string></value><value><string></string></value><value><nil/></value></data></array>",
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.encodeArray(buf, tt.input)

			require.Equal(t, tt.err, err)
			require.Equal(t, tt.expect, buf.String())
		})
	}
}

func Test_encodeBase64(t *testing.T) {
	tests := []struct {
		name   string
		input  []byte
		expect string
		err    error
	}{
		{
			name:   "empty slice",
			input:  []byte{},
			expect: "<base64></base64>",
			err:    nil,
		},
		{
			name: "mixed byte slice",
			input: []byte{
				'a', 'b', 1, 3,
			},
			expect: "<base64>YWIBAw==</base64>",
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.encodeBase64(buf, tt.input)

			require.Equal(t, tt.err, err)
			require.Equal(t, tt.expect, buf.String())
		})
	}
}

func Test_encodeStruct(t *testing.T) {
	ptr := func(v string) *string {
		return &v
	}

	tests := []struct {
		name   string
		input  interface{}
		expect string
		err    error
	}{
		{
			name:   "empty struct",
			input:  struct{}{},
			expect: "<struct></struct>",
			err:    nil,
		},
		{
			name: "no exported fields",
			input: struct {
				unexported string
			}{
				unexported: "I'm unexported",
			},
			expect: "<struct></struct>",
			err:    nil,
		},
		{
			name: "string field",
			input: struct {
				Name string
			}{
				Name: "MyNameIs",
			},
			expect: "<struct><member><name>Name</name><value><string>MyNameIs</string></value></member></struct>",
			err:    nil,
		},
		{
			name: "string pointer field",
			input: struct {
				Name *string
			}{
				Name: ptr("MyNameIs"),
			},
			expect: "<struct><member><name>Name</name><value><string>MyNameIs</string></value></member></struct>",
			err:    nil,
		},

		{
			name: "string pointer field - nil",
			input: struct {
				Name *string
			}{
				Name: nil,
			},
			expect: "<struct><member><name>Name</name><value><nil/></value></member></struct>",
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.encodeStruct(buf, tt.input)

			require.Equal(t, tt.err, err)
			require.Equal(t, tt.expect, buf.String())
		})
	}
}

func Test_encodeTime(t *testing.T) {
	loc := func(name string) *time.Location {
		l, err := time.LoadLocation(name)
		if err != nil {
			return nil
		}

		return l
	}

	tests := []struct {
		name   string
		input  time.Time
		expect string
		err    error
	}{
		{
			name:   "UTC timezone",
			input:  time.Date(2019, 10, 11, 13, 40, 30, 0, time.UTC),
			expect: "<dateTime.iso8601>2019-10-11T13:40:30Z</dateTime.iso8601>",
			err:    nil,
		},

		{
			name:   "Non-UTC timezone",
			input:  time.Date(2019, 10, 11, 13, 40, 30, 0, loc("Europe/Stockholm")),
			expect: "<dateTime.iso8601>2019-10-11T13:40:30+02:00</dateTime.iso8601>",
			err:    nil,
		},

		{
			name:   "Non-UTC timezone",
			input:  time.Date(2019, 10, 11, 13, 40, 30, 0, loc("America/Los_Angeles")),
			expect: "<dateTime.iso8601>2019-10-11T13:40:30-07:00</dateTime.iso8601>",
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.encodeTime(buf, tt.input)

			require.Equal(t, tt.err, err)
			require.Equal(t, tt.expect, buf.String())
		})
	}
}

func Test_encodeMap(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect []string // List of XML fragments that should be present
		err    error
	}{
		{
			name:   "empty map",
			input:  map[string]interface{}{},
			expect: []string{"<struct></struct>"},
			err:    nil,
		},
		{
			name: "map with basic types",
			input: map[string]interface{}{
				"string": "value",
				"int":    42,
				"bool":   true,
				"float":  3.14,
			},
			expect: []string{
				"<member><name>string</name><value><string>value</string></value></member>",
				"<member><name>int</name><value><int>42</int></value></member>",
				"<member><name>bool</name><value><boolean>1</boolean></value></member>",
				"<member><name>float</name><value><double>3.140000</double></value></member>",
			},
			err: nil,
		},
		{
			name: "map with nested structures",
			input: map[string]interface{}{
				"nested": map[string]interface{}{
					"key": "value",
				},
				"array": []string{"a", "b", "c"},
			},
			expect: []string{
				"<member><name>nested</name><value><struct><member><name>key</name><value><string>value</string></value></member></struct></value></member>",
				"<member><name>array</name><value><array><data><value><string>a</string></value><value><string>b</string></value><value><string>c</string></value></data></array></value></member>",
			},
			err: nil,
		},
		{
			name: "map with non-string keys",
			input: map[int]string{
				1: "one",
				2: "two",
			},
			expect: []string{
				"<member><name>1</name><value><string>one</string></value></member>",
				"<member><name>2</name><value><string>two</string></value></member>",
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.encodeMap(buf, tt.input)
			require.Equal(t, tt.err, err)

			output := buf.String()
			// Verify that the output starts with <struct> and ends with </struct>
			require.True(t, strings.HasPrefix(output, "<struct>"))
			require.True(t, strings.HasSuffix(output, "</struct>"))

			// Check that each expected XML fragment is present in the output
			for _, expected := range tt.expect {
				require.Contains(t, output, expected)
			}
		})
	}
}
