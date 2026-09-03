# XML-RPC Client for Go

This is an implementation of client-side part of XML-RPC protocol in Go.

![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/alexejk/go-xmlrpc/build.yml?branch=master)
[![codecov](https://codecov.io/gh/alexejk/go-xmlrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/alexejk/go-xmlrpc)
[![Go Report Card](https://goreportcard.com/badge/alexejk.io/go-xmlrpc)](https://goreportcard.com/report/alexejk.io/go-xmlrpc)

[![GoDoc](https://godoc.org/alexejk.io/go-xmlrpc?status.svg)](https://godoc.org/alexejk.io/go-xmlrpc)
![GitHub](https://img.shields.io/github/license/alexejk/go-xmlrpc)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/alexejk/go-xmlrpc)


## Usage

Add dependency to your project:

```shell
go get -u alexejk.io/go-xmlrpc
```

Use it by creating an `*xmlrpc.Client` and firing RPC method calls with `Call()`.

```go
package main

import(
    "fmt"

    "alexejk.io/go-xmlrpc"
)

func main() {
    client, _ := xmlrpc.NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi")
    defer client.Close()
	
    result := &struct {
        BugzillaVersion struct {
            Version string
        }
    }{}

    _ = client.Call("Bugzilla.version", nil, result)
    fmt.Printf("Version: %s\n", result.BugzillaVersion.Version)
}
```

Customization is supported by passing a list of `Option` to the `NewClient` function. 
For instance:

 - To customize any aspect of `http.Client` used to perform requests, use `HttpClient` option, otherwise `http.DefaultClient` will be used
 - To pass custom headers, make use of `Headers` option.
 - To not fail parsing when unmapped fields exist in RPC responses, use `SkipUnknownFields(true)` option (default is `false`)
 - To change how `<dateTime.iso8601>` values are encoded and decoded, use `TimeFormat` option (default is `time.RFC3339`) - see [Time formats](#time-formats)

### Argument encoding

Arguments to the remote RPC method are passed on as a `*struct`. This struct is encoded into XML-RPC types based on following rules:

* Order of fields in struct type matters - fields are taken in the order they are defined on the **type**.
* Numbers are to be specified as `int` (encoded as `<int>`) or `float64` (encoded as `<double>`)
* Both pointer and value references are accepted (pointers are followed to actual values)
* `map[string]any` types are accepted and encoded into `<struct>`

**Shortcut:**  
If a single `<struct>` argument is expected for the RPC method call, it is sometimes more convenient to pass a `map[string]any` as an argument without wrapping into `struct{}`. This `map[string]any` will be encoded into a single `<struct>` argument with `<member>` elements for each key-value pair.
No other key types are supported and neither is it possible to apply this approach with multiple arguments (or other types).

**Order preservation:**  
As per XML-RPC specification, the order of `<member>` elements in `<struct>` is not defined. When using maps, order of members in a struct is undeterministic, thus it is not guaranteed that the order of `<member>` elements will match the order of keys in the map (due to Go not preserving the order of keys).
To preserve the order, use a struct type with fields defined in the desired order (order is inherited from the struct type itself, not the instance).

### Response decoding

Response is decoded following similar rules to argument encoding.

* Order of fields is important.
* Outer struct should contain exported field for each response parameter (it is possible to ignore unknown structs with `SkipUnknownFields` option).
* Structs may contain pointers - they will be initialized if required.
* Structs may be parsed as `map[string]any`, in case struct member names are not known at compile time. Map keys are enforced to `string` type.

#### Character Encoding Support

The library automatically detects and handles character encodings in XML-RPC responses beyond UTF-8, including ISO-8859-1, Windows-1252, and other charsets commonly found in legacy XML-RPC services.

Character encoding is automatically detected from the XML declaration (e.g., `<?xml version="1.0" encoding="ISO-8859-1"?>`), and the response is transparently converted to UTF-8 for Go string handling. This enables seamless interoperability with XML-RPC servers that don't use UTF-8 encoding.

**Note:** This feature relies on the `golang.org/x/net/html/charset` package.

#### Handling of Empty Values

If XML-RPC response contains no value for well-known data-types, it will be decoded into the default "empty" values as per table below:

| XML-RPC Value           | Default Value |
|-------------------------|--------------|
| `<string/>`             | `""` |
| `<int/>`, `<i4/>`       | `0` |
| `<boolean/>`            | `false` |
| `<double/>`             | `0.0` |
| `<dateTime.iso8601/>`   | `time.Time{}` |
| `<base64/>`             | `nil` |
| `<array><data/><array>` | `nil` |

As per XML-RPC specification, `<struct>` may not have an empty list of `<member>` elements, thus no default "empty" value is defined for it.
Similarly, `<array/>` is considered invalid.

### Time formats

The XML-RPC specification defines `dateTime.iso8601` as ISO8601, but implementations disagree in practice - compact and extended forms, present or absent timezone offsets and fractional seconds are all encountered.
By default this library encodes and decodes using `time.RFC3339`. The `TimeFormat` option changes that:

```go
c, err := xmlrpc.NewClient("https://example.com/rpc", xmlrpc.TimeFormat(&xmlrpc.LayoutTimeFormatter{
    // Encode using the compact form from the specification's example: 19980717T14:08:55
    FormatLayout: xmlrpc.LayoutISO8601Compact,
    // Accept any of the commonly encountered forms when decoding
    ParseLayouts: xmlrpc.CommonParseLayouts(),
    // The compact layout carries no offset, so pin both directions to UTC
    FormatLocation: time.UTC,
    ParseLocation:  time.UTC,
}))
```

Fields come in two pairs: `Format*` controls what goes on the wire, `Parse*` what is accepted off it. `FormatLayout` is the single layout used to encode; `ParseLayouts` are tried in order when decoding and default to `FormatLayout` when unset. When you do set `ParseLayouts` it is the complete list - `FormatLayout` is not added for you, so include it if this client should still read back what it writes. A few layout constants are provided:

| Constant                     | Example                     |
|------------------------------|-----------------------------|
| `LayoutISO8601Compact`       | `19980717T14:08:55`         |
| `LayoutISO8601CompactZoned`  | `19980717T14:08:55+0200`    |
| `LayoutISO8601Basic`         | `19980717T140855`           |
| `LayoutISO8601BasicZoned`    | `19980717T140855+0200`      |
| `LayoutISO8601Extended`      | `1998-07-17T14:08:55`       |
| `LayoutISO8601ExtendedZoned` | `1998-07-17T14:08:55+02:00` |

ISO8601 calls the form without separators *basic* and the form with them *extended*. The layout shown in the XML-RPC specification's example is neither - a basic date with an extended time - and is called *compact* here. The specification does not mandate a layout, and leaves timezone assumptions to server documentation, which is why this is configurable at all. A fractional second is accepted on decode even though no layout declares one.

Notes on timezones:

* `ParseLocation` only applies to decoded values whose layout carries no offset - values that do carry one always keep it. Left unset it follows `time.Parse`: offset-less values are UTC, and an offset matching the process timezone yields that location along with its DST rules. Set it to decode into one location regardless of where the process runs.
* `FormatLocation`, when set, converts values to that location before encoding. By default values are encoded in whichever location they carry, which is fine for zoned layouts but a trap for zone-less ones: a non-UTC `time.Time` would be written as its local wall-clock with no offset, and the receiver has no way to know. **Always set `FormatLocation` when `FormatLayout` carries no offset.**

Servers doing something stranger than a fixed set of layouts can be handled by implementing the `TimeFormatter` interface directly:

```go
type TimeFormatter interface {
    FormatTime(t time.Time) string
    ParseTime(value string) (time.Time, error)
}
```

### Field renaming

XML-RPC specification does not necessarily specify any rules for struct's member names. Some services allow struct member names to include characters not compatible with standard Go field naming.
To support these use-cases, it is possible to remap the field by use of struct tag `xmlrpc`. 

For example, if a response value is a struct that looks like this:

```xml
<struct>
    <member>
        <name>stringValue</name>
        <value><string>bar</string></value>
    </member>
    <member>
        <name>2_numeric.Value</name>
        <value><i4>2</i4></value>
    </member>
</struct>
```

it would be impossible to map the second value to a Go struct with a field `2_numeric.Value` as it's not valid in Go.
Instead, we can map it to any valid field as follows:

```go
v := &struct {
    StringValue string
    SecondNumericValue string `xmlrpc:"2_numeric.Value"`
}{}
```

Similarly, request encoding honors `xmlrpc` tags.

## Building

To build this project, simply run `make all`. 
If you prefer building in Docker instead - `make build-in-docker` is your friend.
