# XML-RPC Client for Go

This is an implementation of client-side part of XML-RPC protocol in Go.

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

    result := &struct {
        BugzillaVersion struct {
            Version string
        }
    }{}

    _ = client.Call("Bugzilla.version", nil, result)
    fmt.Printf("Version: %s\n", result.BugzillaVersion.Version)
}
```

If you want to customize any aspect of `http.Client` used to perform requests, use `NewClientWithHttpClient` instead.
By defailt, an `http.DefaultClient` is used.

### Argument encoding

Arguments to the remote RPC method are passed on as a `*struct`. This struct is encoded into XML-RPC types based on following rules:

* Order of fields in struct type matters - fields are taken in the order they are defined on the **type**.
* Numbers are to be specified as `int` (encoded as `<int>`) or `float64` (encoded as `<double>`)
* Both pointer and value references are accepted (pointers are followed to actual values)

### Response decoding

Response is decoded following similar rules to argument encoding.

* Order of fields is important.
* Outer struct should contain exported field for each response parameter.
