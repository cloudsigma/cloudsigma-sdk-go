# CloudSigma SDK for Go

[![Build Status](https://github.com/cloudsigma/cloudsigma-sdk-go/workflows/build/badge.svg)](https://github.com/cloudsigma/cloudsigma-sdk-go/actions?query=workflow%3Abuild)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudsigma/cloudsigma-sdk-go)](https://goreportcard.com/report/github.com/cloudsigma/cloudsigma-sdk-go)
[![GoDoc](https://img.shields.io/badge/pkg.go.dev-doc-blue)](http://pkg.go.dev/github.com/cloudsigma/cloudsigma-sdk-go)

cloudsigma-sdk-go is the official CloudSigma SDK for the Go programming language.


## Installation

```sh
# X.Y.Z is the version you need
go get github.com/cloudsigma/cloudsigma-sdk-go@vX.Y.Z


# for non Go modules usage or latest version
go get github.com/cloudsigma/cloudsigma-sdk-go
```


## Usage

```go
import "github.com/cloudsigma/cloudsigma-sdk-go"
```
Create a new CloudSigma client, then use the exposed services to access
different parts of the CloudSigma API.

### Authentication

Currently, HTTP Basic Authentication is the only method of authenticating
with the API. You can then use your credentials to create a new client:

```go
client := cloudsigma.NewBasicAuthClient("my-user@my-domain.com", "my-secure-password")
```

### Examples

List all servers for the user.
```go
func main() {
  ctx := context.Background()
  client := cloudsigma.NewBasicAuthClient("my-user@my-domain.com", "my-secure-password")

  // list all servers for the authenticated user
  servers, _, err := client.Servers.List(ctx)
}
```


## Contributing

We love pull requests! Please see the [contribution guidelines](.github/CONTRIBUTING.md).


## License

This SDK is distributed under the BSD 3-Clause License, see [LICENSE](LICENSE) for more information.
