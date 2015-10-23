# Devices

Devices are machines which jobs are executed on.

## Settings

These are the settings per device. They can be set in the [configuration](configuration.md) file. The details of the
settings and the defaults are below.

* `host` - Optional hostname or IP for the device. If not present in configuration, the name is used.
* `protocol` - Optional object. Default is of type "ssh" and port 22 inside of ssh object.
  * `type` - Required if protocol present. The only acceptable value currently is "ssh".
  * `ssh` - Required if protocol present.
     * `port` - Required if protocol present. The port to connect to SSH on.
     * `include_cbc_ciphers` - Optional boolean. By default this is false. If true, the `aes128-cbc`, `aes192-cbc`,
       `aes256-cbc`, and `3des-cbc` ciphers will be supported. This is discouraged as CBC ciphers are known to be
       insecure.
* `tags` - Optional collection of tag strings. This allows workers to choose specific devices.
* `credentials` - Required.
  * `user` - The username to login as
  * `pass` - The password to use to login. Currently only username/password authentication is supported. In the future
    other forms may be supported.
* `jobs` - Required collection of jobs to run. Each job can have its own settings that override the jobs settings.