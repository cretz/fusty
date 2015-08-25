# Devices

Devices are machines which jobs are executed on.

## Settings

These are the settings per device. They can be set in the [configuration](configuration.md) file. The details of the
settings and the defaults are below.

* `protocol` - Optional. The string "ssh" is the only accepted value currently. It is also the default.
* `tags` - Optional collection of tag strings. This allows workers to choose specific devices.
* `credentials` - Required.
  * `user` - The username to login as
  * `pass` - The password to use to login. Currently only username/password authentication is supported. In the future
    other forms may be supported.
  * `prompt` - This is the same type of object as the `prompt` setting in the [jobs](jobs.md) documentation. By default
    this is the same prompt for the first job needing to login.
* `jobs` - Required collection of jobs to run. Each job can have its own settings. Currently these are undefined.