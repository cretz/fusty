# Developers

## Building

This application requires [Go](https://golang.org/) to build. In the cloned directory (assuming your ``GOPATH``
environment variable is set to a path), run:

    go get -u ./...
    go build

## Unit Testing

In the cloned directory, run:

    go test ./...

## Contributing

Besides unit and integration tests, all source must run and pass the following:

    go fmt ./...
    go vet ./...
    golint ./...

## Integration Testing

TODO