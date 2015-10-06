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
* `commands` - Array of command types. No default, required if type is `command`. Each command item can contain:
  * `command` - String in each command item for the command to type.
  * `expect` - Optional array of string regex patterns. If any of these patterns is matched, the command is considered a
    success. If none of these is matched the command is considered a failure, even if nothing in `expect_not` matches.
    If `expect_not` is present and anything matches there, the failure emitted trumps any successful match here. If both
    this and `expect_not` are not present the command is always considered a success.
  * `expect_not` - Optional array of string regex patterns. If any of these patterns is matched, the command is
    considered a failure. This is true regardless of whether anything in `expect` matches.
  * `timeout` - The optional amount of time in seconds to wait until something in `expect` or `expect_not` matches. If
    only `expect_not` is present, when the timeout is reached without a match the command is considered a success. If
    `expect` is present (regardless of whether `expect_not` is present), when timeout is reached without a match the
    command is considered a failure. If both `expect` and `expect_not` are not present, the system will wait the given
    amount of time always and always consider the result a success. If this is not present, it is defaulted at 120
    seconds. This value must be at least 1 if `expect` or `expect_not` are present. If neither `expect` nor `expect_not`
    are present, this value can be set to 0 to continue immediately.
  * `implicit_enter` - Optional boolean on whether there is an implicit "enter" that is typed after every command. By
    default this is true.
* `command_generic` - Object that has settings as though they are on each command item detailed in the previous bullet
  point.
* `file` - No default, required if type is `file`. Each key is the fully qualified path. Multiple files will be
  concatenated in alphabetical order.
  * `FILEPATH` - The file path to fetch.
    * `compression` - If present, this is the compression used by the file. Only `gzip` supported currently.

### Prompt Conditions



## Job Distribution

Jobs are distributed across workers on a first-come-first-serve basis. All jobs may run concurrently, even if they are
for the same device. Therefore job configurers are encouraged to avoid mutating or affecting global state which could
affect other jobs on the same device.