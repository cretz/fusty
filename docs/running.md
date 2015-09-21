# Running

## Installing

To install, simply download the archive at (TODO: link here) for the desired operating system. The archive will contain
a single `fusty` binary that can be executed.

The application may be built from source instead of using the binary if desired. See the [developer](developers.md)
documentation for more information.

## Running a Controller

A controller is the main server coordinating the application. To run it, simply execute:

    fusty controller [-config=./fusty.conf.json] [-verbose]

Once this has started, the controller's web application will be available at the configured host and port. Default is
https://127.0.0.1:9400. On first start without a configuration file, the web application will lead you through setup.

The `-config` option can be provided to point to a specific configuration file. If not present, the application will
look for `fusty.conf.json` in the current working directory. See the [configuration](configuration.md) documentation for
more information.

The `-verbose` option can be set to show more log output. Note, these extra-verbose messages currently do not go to
syslog.

## Running a Worker

    fusty worker [-controller=http://127.0.0.1:9400] [-tag=tag1] [-tag=tag2] [-sleep=15] [-maxjobs=N] [-verbose]

A worker doesn't have a configuration file but it does have optional settings:

* `-controller` - This is the base URL for the controller. If not provided, it assumes `http://127.0.0.1:9400`.
* `-tag` - This is the device tag that this worker accepts work for. This can be provided multiple times for multiple
  tags. If not provided, this worker accepts work for all device types.
* `-sleep` - The number of seconds to wait to ask the controller for more work if none was given last request. The
  higher this number is, the more "off" a job run may be. By default this is 15 seconds.
* `-maxjobs` - The maximum number of jobs this worker can be executing at any one time. By default this is 2000.
* `-timeout` - The maximum number of seconds to wait for the controller to respond to HTTP. By default this is 3.
* `-verbose` - If set, the log output will be verbose. Note, these extra-verbose messages currently do not go to syslog.

In the future, there will also be settings for TLS configuration.