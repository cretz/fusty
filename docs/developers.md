# Developers

## Environment

This should work with the latest version of [Go](https://golang.org/). Based on Go rules, the checked out path needs to
be in a certain location under the `GOPATH` environment variable in order to function properly.

Assuming that `GOPATH` is set to `foo/bar`, then the repository needs to be checked out at
`foo/bar/src/gitlab.com/cretz/fusty`. This allows dependencies to be placed properly in their place.

## Building

Assuming the environment is set up as above, run this at the repo root (i.e. `$GOPATH/src/gitlab.com/cretz/fusty`):

    go get -u ./...
    go build

This will place an executable called `fusty` in that directory.

## Unit Testing

In the cloned directory, run:

    go test ./...

Currently there are no unit tests, only integration tests

## Contributing

Besides unit and integration tests, all source must run and pass the following:

    go fmt ./...
    go vet ./...
    golint ./...

## Integration Testing

This uses [GoConvey](https://github.com/smartystreets/goconvey). Simply run the following:

    go get github.com/smartystreets/goconvey

Then there is a binary at `$GOPATH/bin` called `goconvey`. Simply run it in the `fusty` directory and the results can
be viewed at http://localhost:8080.

Note, the integration tests currently work by executing the binary from the outside. This means that the "`go build`"
has to be executed on every change manually. In the future this will be automated.

Note, the integration tests currently work by executing the binary from the outside. This means that code coverage
information inside the binary is unavailable. Programmatically invoking Fusty is a possible future feature to alleviate
this.