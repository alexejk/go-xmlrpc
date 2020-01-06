# XML-RPC Client for Go

This is an implementation of client-side part of XML-RPC protocol in Go.

## Usage

```go
client, _ := NewClient("https://bugzilla.mozilla.org/xmlrpc.cgi")

resp := &struct {
    BugzillaVersion struct {
        Version string
    }
}{}

_ = c.Call("Bugzilla.version", nil, resp)
fmt.Printf("Version: %s\n", result.BugzillaVersion.Version)
```

If you want to customize any aspect of `http.Client` used to perform requests, use `NewClientWithHttpClient` instead.
By defailt, an `http.DefaultClient` is used.

### Argument encoding

Arguments to the remote RPC method are passed on as a `*struct`. This struct is encoded into XML-RPC types based on following rules:

* Order of fields in struct type matters - fields are taken in the order they are defined on the **type**.
* Numbers are to be specified as `int` (encoded as `<int>`) or `float64` (encoded as `<double>`)

### Response decoding
