# Integration Testing

## Emulated Environments

### GNS3 VM

Currently there is limited support for [GNS3](http://www.gns3.com/) environments on Vagrant. Follow the following steps
inside of the `emulated` subfolder:

1. Make sure [Packer](https://www.packer.io/), [VirtualBox](https://www.virtualbox.org/), and
   [Vagrant](https://www.vagrantup.com/) are installed
2. Download the latest VirtualBox VM at https://github.com/GNS3/gns3-vm/releases (`0.9.6` at the time of this writing)
3. Set the following environment variables:
   * `GNS3_VERSION` - Latest version of GNS to use (`1.3.9` is latest stable as of this writing)
   * `GNS3_UPDATE_FLAVOR` - Either `stable` or `testing` (former is recommended)
   * `GNS3_SRC` - Path to the downloaded OVA file in the previous step
4. Run `packer build -only virtualbox-ovf gns3_release.json` which will create `packer_virtualbox-ovf_virtualbox.box`
   in the `emulated` folder
5. Run `vagrant up` to start the VM

Now the GNS3 VM is running and can be SSH'd into with the user/pass of `gns3`/`gns3` on port 2222. Note, the
`gns3_release.json` file is basically exactly what is in the [GNS3 VM repository](https://github.com/GNS3/gns3-vm) but
with a Vagrant post processor configured.

### Arista VM

TODO

## Running

Assuming the emulated environments are running, the integration tests can be executed. They use
[GoConvey](https://github.com/smartystreets/goconvey). Simply run the following:

    go get github.com/smartystreets/goconvey

Then there is a binary at `$GOPATH/bin` called `goconvey`. Simply run it in the `fusty` directory and the results can
be viewed at http://localhost:8080.

Note, the integration tests currently work by executing the binary from the outside. This means that the "`go build`"
has to be executed on every change manually. In the future this will be automated.

Note, the integration tests currently work by executing the binary from the outside. This means that code coverage
information inside the binary is unavailable. Programmatically invoking Fusty is a possible future feature to alleviate
this.