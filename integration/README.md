# Integration Testing

## Emulated Environments

### Linux VM

How to start the VM:

    vagrant up linux-vm

This must be started to run heavy integration tests.

## Running

Note, the integration tests currently work by executing the binary from the outside. This means that the "`go build`"
has to be executed on every change manually. In the future this will be automated.

### Lightweight Integration Tests

Some integration tests are lightweight and can be continuously executed via
[GoConvey](https://github.com/smartystreets/goconvey). Simply run the following:

    go get github.com/smartystreets/goconvey

Then there is a binary at `$GOPATH/bin` called `goconvey`. Simply run it in the `fusty` directory and the results can
be viewed at http://localhost:8080.

Note, the integration tests currently work by executing the binary from the outside. This means that code coverage
information inside the binary is unavailable. Programmatically invoking Fusty is a possible future feature to alleviate
this.

### Heavy Integration Tests

Some tests are a bit heavier. This includes tests that require emulated environments to be online. These must be
executed from withing this `integration` folder. Once in that folder, simply run:

    go test -tags heavy