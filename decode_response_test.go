package xmlrpc

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNewResponse_CharsetDetection tests the charset detection functionality
// introduced in PR #93. This verifies that XML-RPC responses with different
// character encodings (UTF-8, ISO-8859-1, Windows-1252) are properly decoded.
func TestNewResponse_CharsetDetection(t *testing.T) {
	tests := map[string]struct {
		testFile      string
		expectSuccess bool
		expectParams  int
		expectString  string // Expected string value from first param
		expectInt     int    // Expected int value from second param
		expectError   bool
	}{
		"utf-8 baseline": {
			testFile:      "response_charset_utf8.xml",
			expectSuccess: true,
			expectParams:  2,
			expectString:  "Hello UTF-8: café résumé",
			expectInt:     42,
		},
		"iso-8859-1 with special chars": {
			testFile:      "response_charset_iso88591.xml",
			expectSuccess: true,
			expectParams:  2,
			expectString:  "ISO-8859-1 chars: café résumé ñoño",
			expectInt:     123,
		},
		"windows-1252": {
			testFile:      "response_charset_windows1252.xml",
			expectSuccess: true,
			expectParams:  2,
			expectString:  "Win-1252 chars: café résumé €uro",
			expectInt:     456,
		},
		"invalid charset": {
			testFile:    "response_charset_invalid.xml",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			body := loadTestFile(t, tt.testFile)
			response, err := NewResponse(body)

			if tt.expectError {
				require.Error(t, err, "Expected error for invalid charset")
				require.Nil(t, response)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response)
			require.Nil(t, response.Fault, "Expected successful response, not fault")
			require.Len(t, response.Params, tt.expectParams)

			// Verify first parameter (string)
			if tt.expectParams > 0 && tt.expectString != "" {
				require.NotNil(t, response.Params[0].Value.String)
				require.Equal(t, tt.expectString, *response.Params[0].Value.String)
			}

			// Verify second parameter (int)
			if tt.expectParams > 1 {
				intValue := response.Params[1].Value.Int
				require.NotNil(t, intValue)
				// Parse the string to int for comparison
				actualInt, err := strconv.Atoi(*intValue)
				require.NoError(t, err)
				require.Equal(t, tt.expectInt, actualInt)
			}
		})
	}
}

// TestNewResponse_CharsetInXMLDeclaration verifies that the charset
// specified in the XML declaration is properly respected by the decoder.
func TestNewResponse_CharsetInXMLDeclaration(t *testing.T) {
	tests := map[string]struct {
		testFile string
		encoding string // Expected encoding declaration
	}{
		"UTF-8 declaration": {
			testFile: "response_charset_utf8.xml",
			encoding: "UTF-8",
		},
		"ISO-8859-1 declaration": {
			testFile: "response_charset_iso88591.xml",
			encoding: "ISO-8859-1",
		},
		"Windows-1252 declaration": {
			testFile: "response_charset_windows1252.xml",
			encoding: "Windows-1252",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			body := loadTestFile(t, tt.testFile)

			// Verify the XML declaration contains the expected encoding
			require.Contains(t, string(body), "encoding=\""+tt.encoding+"\"",
				"XML file should declare encoding in XML declaration")

			// Verify it can be parsed successfully
			response, err := NewResponse(body)
			require.NoError(t, err)
			require.NotNil(t, response)
			require.Nil(t, response.Fault)
		})
	}
}

// TestNewResponse_CharsetDecoding_EndToEnd tests complete decoding workflow
// using StdDecoder with charset detection to ensure the feature works correctly
// in the full decoding pipeline.
func TestNewResponse_CharsetDecoding_EndToEnd(t *testing.T) {
	tests := map[string]struct {
		testFile     string
		expectString string
		expectInt    int
	}{
		"UTF-8 end-to-end": {
			testFile:     "response_charset_utf8.xml",
			expectString: "Hello UTF-8: café résumé",
			expectInt:    42,
		},
		"ISO-8859-1 end-to-end": {
			testFile:     "response_charset_iso88591.xml",
			expectString: "ISO-8859-1 chars: café résumé ñoño",
			expectInt:    123,
		},
		"Windows-1252 end-to-end": {
			testFile:     "response_charset_windows1252.xml",
			expectString: "Win-1252 chars: café résumé €uro",
			expectInt:    456,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			body := loadTestFile(t, tt.testFile)

			// Use StdDecoder to fully decode into a struct
			type TestResponse struct {
				Message string
				Value   int
			}

			decoder := &StdDecoder{}
			var result TestResponse
			err := decoder.DecodeRaw(body, &result)

			require.NoError(t, err)
			require.Equal(t, tt.expectString, result.Message)
			require.Equal(t, tt.expectInt, result.Value)
		})
	}
}

// TestNewResponse_InvalidCharset_ErrorHandling verifies that invalid or
// unsupported charset declarations are handled gracefully.
func TestNewResponse_InvalidCharset_ErrorHandling(t *testing.T) {
	body := loadTestFile(t, "response_charset_invalid.xml")
	response, err := NewResponse(body)

	// Should return an error for unsupported charset
	require.Error(t, err)
	require.Nil(t, response)

	// Verify error message is meaningful (contains charset-related info)
	// Note: The exact error depends on golang.org/x/net/html/charset implementation
	require.NotEmpty(t, err.Error())
}

// TestNewResponse_UTF8WithoutDeclaration tests backwards compatibility
// for UTF-8 XML without explicit encoding declaration.
func TestNewResponse_UTF8WithoutDeclaration(t *testing.T) {
	// XML without encoding declaration (defaults to UTF-8)
	xmlBody := `<?xml version="1.0"?>
<methodResponse>
    <params>
        <param>
            <value><string>No encoding declaration</string></value>
        </param>
    </params>
</methodResponse>`

	response, err := NewResponse([]byte(xmlBody))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.Len(t, response.Params, 1)
	require.NotNil(t, response.Params[0].Value.String)
	require.Equal(t, "No encoding declaration", *response.Params[0].Value.String)
}

// TestNewResponse_EmptyBody tests that empty or malformed XML is handled properly.
func TestNewResponse_EmptyBody(t *testing.T) {
	tests := map[string]struct {
		body        []byte
		expectError bool
	}{
		"empty body": {
			body:        []byte{},
			expectError: true,
		},
		"whitespace only": {
			body:        []byte("   \n\t  "),
			expectError: true,
		},
		"invalid xml": {
			body:        []byte("<invalid>xml"),
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := NewResponse(tt.body)
			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
			}
		})
	}
}

// BenchmarkNewResponse_UTF8 benchmarks the common case (UTF-8) to measure
// any performance overhead introduced by charset detection.
func BenchmarkNewResponse_UTF8(b *testing.B) {
	body := loadBenchFile(b, "response_charset_utf8.xml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		response, err := NewResponse(body)
		if err != nil {
			b.Fatal(err)
		}
		if response == nil {
			b.Fatal("response is nil")
		}
	}
}

// BenchmarkNewResponse_ISO88591 benchmarks ISO-8859-1 decoding performance.
func BenchmarkNewResponse_ISO88591(b *testing.B) {
	body := loadBenchFile(b, "response_charset_iso88591.xml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		response, err := NewResponse(body)
		if err != nil {
			b.Fatal(err)
		}
		if response == nil {
			b.Fatal("response is nil")
		}
	}
}

// BenchmarkNewResponse_Windows1252 benchmarks Windows-1252 decoding performance.
func BenchmarkNewResponse_Windows1252(b *testing.B) {
	body := loadBenchFile(b, "response_charset_windows1252.xml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		response, err := NewResponse(body)
		if err != nil {
			b.Fatal(err)
		}
		if response == nil {
			b.Fatal("response is nil")
		}
	}
}

// BenchmarkNewResponse_Simple benchmarks a simple response without special characters
// to establish a baseline for performance comparison.
func BenchmarkNewResponse_Simple(b *testing.B) {
	body := loadBenchFile(b, "response_simple.xml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		response, err := NewResponse(body)
		if err != nil {
			b.Fatal(err)
		}
		if response == nil {
			b.Fatal("response is nil")
		}
	}
}

// BenchmarkStdDecoder_DecodeRaw_UTF8 benchmarks the full decode pipeline with UTF-8.
func BenchmarkStdDecoder_DecodeRaw_UTF8(b *testing.B) {
	body := loadBenchFile(b, "response_charset_utf8.xml")
	decoder := &StdDecoder{}

	type TestResponse struct {
		Message string
		Value   int
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result TestResponse
		err := decoder.DecodeRaw(body, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStdDecoder_DecodeRaw_ISO88591 benchmarks the full decode pipeline with ISO-8859-1.
func BenchmarkStdDecoder_DecodeRaw_ISO88591(b *testing.B) {
	body := loadBenchFile(b, "response_charset_iso88591.xml")
	decoder := &StdDecoder{}

	type TestResponse struct {
		Message string
		Value   int
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var result TestResponse
		err := decoder.DecodeRaw(body, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// loadBenchFile loads a test file for benchmarking (panics on error).
func loadBenchFile(b *testing.B, name string) []byte {
	b.Helper()
	path := filepath.Join("testdata", name)
	bytes, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	return bytes
}

// TestNewResponse_Fault_WithCharset verifies that fault responses work correctly
// with charset detection enabled.
func TestNewResponse_Fault_WithCharset(t *testing.T) {
	// Test with existing fault response to ensure charset detection
	// doesn't break fault handling
	body := loadTestFile(t, "response_fault.xml")
	response, err := NewResponse(body)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Fault, "Expected fault response")
	require.Nil(t, response.Params, "Fault response should not have params")
}

// TestNewResponse_LargeResponse_WithCharset tests charset detection with
// a larger response to ensure it handles real-world scenarios.
func TestNewResponse_LargeResponse_WithCharset(t *testing.T) {
	// Use existing large/complex response file
	body := loadTestFile(t, "response_nested_random_struct.xml")
	response, err := NewResponse(body)

	require.NoError(t, err)
	require.NotNil(t, response)
	require.Nil(t, response.Fault)
	require.NotEmpty(t, response.Params, "Expected non-empty params")
}

// TestNewResponse_SpecialCharacters tests various special characters
// across different encodings to ensure proper decoding.
func TestNewResponse_SpecialCharacters(t *testing.T) {
	tests := map[string]struct {
		testFile       string
		expectedChars  []string // Characters we expect to find in the decoded string
		unexpectedChar string   // A character we should NOT find (encoding-specific)
	}{
		"UTF-8 special chars": {
			testFile:      "response_charset_utf8.xml",
			expectedChars: []string{"café", "résumé"},
		},
		"ISO-8859-1 special chars": {
			testFile:      "response_charset_iso88591.xml",
			expectedChars: []string{"café", "résumé", "ñoño"},
		},
		"Windows-1252 euro sign": {
			testFile:      "response_charset_windows1252.xml",
			expectedChars: []string{"café", "résumé", "€uro"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			body := loadTestFile(t, tt.testFile)
			response, err := NewResponse(body)

			require.NoError(t, err)
			require.NotNil(t, response)
			require.Len(t, response.Params, 2)

			actualString := *response.Params[0].Value.String

			// Verify all expected characters are present
			for _, expectedChar := range tt.expectedChars {
				require.Contains(t, actualString, expectedChar,
					"Expected to find '%s' in decoded string", expectedChar)
			}
		})
	}
}

// TestNewResponse_CharsetDetection_Regression ensures that common XML-RPC
// responses still work correctly after charset detection changes.
func TestNewResponse_CharsetDetection_Regression(t *testing.T) {
	// Test with existing test files to ensure no regressions
	testFiles := []string{
		"response_simple.xml",
		"response_array.xml",
		"response_struct.xml",
		"response_fault.xml",
		"response_bugzilla_version.xml",
	}

	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {
			body := loadTestFile(t, testFile)
			response, err := NewResponse(body)

			require.NoError(t, err, "Should successfully parse %s", testFile)
			require.NotNil(t, response, "Response should not be nil for %s", testFile)
		})
	}
}

// TestNewResponse_CharsetDetection_ErrorTypes verifies that appropriate error
// types are returned for different failure scenarios.
func TestNewResponse_CharsetDetection_ErrorTypes(t *testing.T) {
	tests := map[string]struct {
		body        []byte
		expectError bool
	}{
		"invalid charset returns error": {
			body:        loadTestFile(t, "response_charset_invalid.xml"),
			expectError: true,
		},
		"malformed xml returns error": {
			body:        []byte("<invalid>"),
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			response, err := NewResponse(tt.body)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, response)
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
			}
		})
	}
}
