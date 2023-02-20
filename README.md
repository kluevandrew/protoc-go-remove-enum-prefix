# protoc-go-remove-enum-prefix
[![Tests][gh-test-actions-badge]][gh-actions-url]
[![Lint][gh-lint-actions-badge]][gh-actions-url]

## Why?

Golang [protobuf](https://github.com/golang/protobuf) adds type prefix to generated constants that makes
code harder to read and follow with longer enums (both longer type name and longer enum values).
See [protoc-gen-go: remove type name from generated enum](https://github.com/golang/protobuf/issues/513).
This tool removes that prefix in generated `*.pb.go`.

## Special thanks

Special thanks to [Diep Pham](https://github.com/favadi), code of that util based on his [protoc-go-inject-tag](https://github.com/favadi/protoc-go-inject-tag).

## Install

- [protobuf version 3](https://github.com/google/protobuf)

  For OS X:

  ```console
  brew install protobuf
  ```

- go support for protobuf: `go get -u github.com/golang/protobuf/{proto,protoc-gen-go}`

- `go install github.com/kluevandrew/protoc-go-remove-enum-prefix` or download the
  binaries from the releases page.

## Usage

```console
$ protoc-go-remove-enum-prefix -h
Usage of protoc-go-remove-enum-prefix:
  -input string
        pattern to match input file(s)
  -verbose
        verbose logging
```

Add a comment with the following syntax before enums, and prefixes on all associated constants in resulting `.pb.go` file would be removed.

```proto
// @go-enum-no-prefix
```

## Example

```proto
// file: test.proto
syntax = "proto3";

package pb;
option go_package = "/pb";

// @go-enum-no-prefix
enum ExampleEnum {
  EXAMPLE_ENUM_UNSPECIFIED = 0;
  EXAMPLE_ENUM_ONE = 1;
  EXAMPLE_ENUM_TWO = 2;
}

message ExampleMessage {
  string id = 1;
  SomeEnum type = 2;
}
```

Generate your `.pb.go` files with the protoc command as normal:

```console
protoc --proto_path=. --go_out=paths=source_relative:. test.proto
```

Then run `protoc-go-remove-enum-prefix` against the generated files (e.g `test.pb.go`):

```console
$ protoc-go-remove-enum-prefix -input=./test.pb.go
# or
$ protoc-go-remove-enum-prefix -input="*.pb.go"
```

The custom tags will be injected to `test.pb.go`:

```go
type SomeEnum int32

const (
  SOME_ENUM_UNSPECIFIED SomeEnum = 0
  SOME_ENUM_ONE         SomeEnum = 1
  SOME_ENUM_TWO         SomeEnum = 2
)
```

[gh-lint-actions-badge]: https://github.com/kluevandrew/protoc-go-remove-enum-prefix/actions/workflows/lint.yml/badge.svg
[gh-test-actions-badge]: https://github.com/kluevandrew/protoc-go-remove-enum-prefix/actions/workflows/test.yml/badge.svg
[gh-actions-url]: https://github.com/kluevandrew/protoc-go-remove-enum-prefix/actions

