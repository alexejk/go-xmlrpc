## 0.4.1

Bugfixes:

* Adding missing handling of undeclared value types to default to `string` as per XML-RPC spec (previously `nil` would be returned)

Library is now built against Go 1.21

## 0.4.0

Improvements:

* Ability to remap struct member names to via `xmlrpc` tags (#47)
* Ability to skip unknown fields by `SkipUnknownFields(bool)` `Option`. Default is still `false` (#48)

Library is now built against Go 1.19

## 0.3.0

Improvements:

* Fixes go routine leak that is caused by `Codec` (#52)
* A bit more robust tests that do not call remote systems
* House keeping: dependency updates, no longer using deprecated methods in Go, making linter happier..

Library is now built against Go 1.18

## 0.2.0

Improvements:

* `NewClient` supports receiving a list of `Option`s that modify clients behavior.  
Initial supported options are:

  * `HttpClient(*http.Client)` - set custom `http.Client` to be used
  * `Headers(map[string]string)` - set custom headers to use in every request (kudos: @Nightapes)
  * `UserAgent(string)` - set User-Agent identification to be used (#6). This is a shortcut for just setting `User-Agent` custom header

Deprecations:

* `NewCustomClient` is deprecated in favor of `NewClient(string, ...Option)` with `HttpClient(*http.Client)` option. 
This method will be removed in future versions.

## 0.1.2

Improvements to parsing logic for responses:

* If response struct members are in snake-case - Go struct should have member in camel-case
* It is now possible to use type aliases when decoding a response (#1)
* It is now possible to use pointers to fields, without getting an error (#2)

## 0.1.1

Mainly documentation and internal refactoring:

* Made `Encoder` and `Decoder` into interfaces with corresponding `StdEncoder` / `StdDecoder`.
* Removal of intermediate objects in `Codec`

## 0.1.0

Initial release version of XML-RPC client.

* Support for all XML-RPC types both encoding and decoding.
* A client implementation based on `net/rpc` for familiar interface.
* No external dependencies (except testing code dependencies)
