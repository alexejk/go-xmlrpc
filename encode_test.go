package xmlrpc

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdEncoder_Encode(t *testing.T) {
	tests := []struct {
		name       string
		methodName string
		args       interface{}
		expect     string
		err        error
	}{
		{
			name:       "No args",
			methodName: "myMethod",
			args:       nil,
			expect:     `<methodCall><methodName>myMethod</methodName></methodCall>`,
			err:        nil,
		},
		{
			name:       "Args empty struct as pointer",
			methodName: "myMethod",
			args:       &struct{}{},
			expect:     `<methodCall><methodName>myMethod</methodName></methodCall>`,
			err:        nil,
		},
		{
			name:       "Args empty struct as value",
			methodName: "myMethod",
			args:       struct{}{},
			expect:     `<methodCall><methodName>myMethod</methodName></methodCall>`,
			err:        nil,
		},
		{
			name:       "Args as pointer",
			methodName: "myMethod",
			args: &struct {
				String string
			}{
				String: "my-name",
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><string>my-name</string></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "Args as value",
			methodName: "myMethod",
			args: struct {
				String string
			}{
				String: "my-name",
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><string>my-name</string></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "Args with unexported fields",
			methodName: "myMethod",
			args: struct {
				smthUnexported string
			}{
				smthUnexported: "i-am-unexported",
			},
			expect: `<methodCall><methodName>myMethod</methodName></methodCall>`,
			err:    nil,
		},
		{
			name:       "Boolean args",
			methodName: "myMethod",
			args: &struct {
				BooleanTrue  bool
				BooleanFalse bool
			}{
				// Order purposely swapped
				BooleanFalse: false,
				BooleanTrue:  true,
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><boolean>1</boolean></value></param><param><value><boolean>0</boolean></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "Numerical args",
			methodName: "myMethod",
			args: &struct {
				Int    int
				Double float64
			}{
				Int:    123,
				Double: float64(12345),
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><int>123</int></value></param><param><value><double>12345.000000</double></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "String arg - simple",
			methodName: "myMethod",
			args: &struct {
				String string
			}{
				String: "my-name",
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><string>my-name</string></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "String arg - encoded",
			methodName: "myMethod",
			args: &struct {
				String string
			}{
				String: `<div class="whitespace">&nbsp;</div>`,
			},
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><string>&lt;div class=&#34;whitespace&#34;&gt;&amp;nbsp;&lt;/div&gt;</string></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "Struct args - encoded",
			methodName: "myMethod",
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
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><struct><member><name>String</name><value><string>foo</string></value></member></struct></value></param></params></methodCall>`,
			err:    nil,
		},
		{
			name:       "Struct args renamed - encoded",
			methodName: "myMethod",
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
			expect: `<methodCall><methodName>myMethod</methodName><params><param><value><struct><member><name>2-.Arg</name><value><string>foo</string></value></member></struct></value></param></params></methodCall>`,
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(strings.Builder)
			enc := &StdEncoder{}
			err := enc.Encode(buf, tt.methodName, tt.args)
			require.Equal(t, tt.expect, buf.String())
			require.Equal(t, tt.err, err)
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

// XMLRPCValue represents a value in XML-RPC format
type XMLRPCValue struct {
	Type  string      `xml:"type,attr"`
	Value interface{} `xml:",chardata"`
}

// XMLRPCMember represents a member in XML-RPC struct
type XMLRPCMember struct {
	Name  string      `xml:"name"`
	Value XMLRPCValue `xml:"value"`
}

// XMLRPCStruct represents an XML-RPC struct
type XMLRPCStruct struct {
	Members []XMLRPCMember `xml:"member"`
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
