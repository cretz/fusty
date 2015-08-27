# Jobs

Jobs are executed on an interval in Fusty.

## Settings

These are the settings per job. They can be set in the [configuration](configuration.md) file. The details of the
settings and the defaults are below.

* `prompt` - No default, required
  * `ends_with` - If present, this means that the prompt is reached when the end of the input is this string
* `schedule` - No default, one of three formats required
  * `cron` - [Cron-formatted](https://en.wikipedia.org/wiki/Cron#Format) string
  * `duration` - Simple duration string in the form of "number timeunit". The number must be a whole number and time
    unit can have a trailing "s" or not. The durations cannot have units greater than days. All intervals are aligned to
    1970-01-01.
  * `iso_8601` - [ISO-8601](https://en.wikipedia.org/wiki/ISO_8601#Time_intervals) interval string. This is expected to
    be a repeating interval.
  * `fixed` - Unix time to run this exactly
* `command` - No default, required
  * `inline` - An array of commands to run to obtain the text to backup

## Job Distribution

Jobs are distributed across workers on a first-come-first-serve basis. If multiple jobs for the same same device are
waiting to go to the next worker for work and they use the same protocol, they will be executed together. This allows
jobs to share the same connections and authentications for multiple jobs if the jobs are configured for the same
frequency.