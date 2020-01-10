## 0.1.3 (WIP)

Improvements:

* User-Agent can now be configured on the Client (#6)

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
