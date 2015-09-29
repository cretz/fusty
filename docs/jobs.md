# Jobs

Jobs are executed on an interval in Fusty.

## Settings

These are the settings per job. They can be set in the [configuration](configuration.md) file. The details of the
settings and the defaults are below.

* `schedule` - No default, one of three formats required
  * `cron` - [Cron-formatted](https://en.wikipedia.org/wiki/Cron#Format) string. Note, Fusty supports second-level
    precision as an optional first value of the cron string. E.g. this runs every 45 seconds: `*/45 * * * * * *`.
  * `duration` - Simple duration string in the form of "number timeunit". The number must be a whole number and time
    unit can have a trailing "s" or not. The durations cannot have units greater than days. All intervals are aligned to
    1970-01-01.
  * `iso_8601` - [ISO-8601](https://en.wikipedia.org/wiki/ISO_8601#Time_intervals) interval string. This is expected to
    be a repeating interval.
  * `fixed` - Unix time to run this exactly
* `type` - Optional job type. Default is `command` but can also be `file`.
* `command` - No default, required if type is `command`
  * `inline` - An array of commands to run to obtain the text to backup
* `file` - No default, required if type is `file`. Each key is the fully qualified path. Multiple files will be
  concatenated in alphabetical order.
  * `FILEPATH` - The file path to fetch.
    * `compression` - If present, this is the compression used by the file. Only `gzip` supported currently.

## Job Distribution

Jobs are distributed across workers on a first-come-first-serve basis. All jobs may run concurrently, even if they are
for the same device. Therefore job configurers are encouraged to avoid mutating or affecting global state which could
affect other jobs on the same device.