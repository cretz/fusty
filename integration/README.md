# Integration Testing

## Emulated Environments

### GNS3 VM

Currently there is limited support for [GNS3](http://www.gns3.com/) environments on Vagrant. Perform the following steps
in the `emulated` directory to create the VM:

1. Make sure [Packer](https://www.packer.io/), [VirtualBox](https://www.virtualbox.org/), and
   [Vagrant](https://www.vagrantup.com/) are installed
2. Download the latest VirtualBox VM at https://github.com/GNS3/gns3-vm/releases (`0.9.6` at the time of this writing)
3. Set the following environment variables:
   * `GNS3_VERSION` - Latest version of GNS to use (`1.3.9` is latest stable as of this writing)
   * `GNS3_UPDATE_FLAVOR` - Either `stable` or `testing` (former is recommended)
   * `GNS3_SRC` - Path to the downloaded OVA file in the previous step
4. Run `packer build -only virtualbox-ovf gns3_release.json` which will create `packer_virtualbox-ovf_virtualbox.box`
   in the `emulated` folder

Now to start the VM:

    vagrant up gns3-vm

Now the GNS3 VM is running and can be SSH'd into with the user/pass of `gns3`/`gns3` on port 2222. Note, the
`gns3_release.json` file is basically exactly what is in the [GNS3 VM repository](https://github.com/GNS3/gns3-vm) but
with a Vagrant post processor configured. Nothing has been done with this VM yet, but there might be in the future.

### Arista VM

This VM is created similar to the GNS3 one. Perform the following steps in the `emulated` directory to create the VM:

1. Make sure [Packer](https://www.packer.io/), [VirtualBox](https://www.virtualbox.org/), and
   [Vagrant](https://www.vagrantup.com/) are installed and on the `PATH` properly. Due to
   [some issues](https://github.com/mitchellh/vagrant/issues/6120), make sure the VirtualBox version is at least 5.0.3.
2. Make sure the git submodule in `packer-veos` is checked out
3. Download `vEOS-lab-[release].vmdk` from [here](https://www.arista.com/en/support/software-download)(yes,
   registration/login required, sorry) and name it `vEOS.vmdk`
4. Same as above for `Aboot-[release].iso` and rename it `Aboot-vEOS.iso`
5. Run `go run main.go build-arista-vm`

Now to start the VM:
                                       
    vagrant up arista-vm

Note, there is currently a bug preventing this from working properly.

### Juniper VM

The Juniper VM is already an existing Vagrant box. Therefore, to start it, simply run the following in the `emulated`
directory:

    vagrant up juniper-vsrx

This VM is actually used in the integration tests.

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