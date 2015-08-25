# Configuration

Fusty is configured via configuration files. The configuration can currently only be written in JSON. In the future
[YAML](http://yaml.org/), [TOML](https://github.com/toml-lang/toml), [HCL](https://github.com/hashicorp/hcl), and
JSON with comments may be supported.

By default the configuration file is assumed to be `fusty.conf.json` in the current working directory, but this can be
configured to point to any path. Each section below represents one section of configuration, but they are all together
in the single configuration file. Although the examples in the sections below use JSON with comments, the actual
implementation does not currently support comments in JSON files. Some sections are commented out because they are
optional.

In the future, the web application may also be able to handle configuration in addition to manual file editing.

## General Settings

Below is a JSON configuration for general top-level settings for Fusty. Comments are present to explain each part:

```json

// The IP to listen on. Default is all IPs
// "ip": "127.0.0.1"

// The port to listen on. Default is 9400
// "port": 9400

// The log level. Default is info
// "log_level": "info"

// Set true to log to syslog in addition to stdout. Fails on Windows. Default is false
// "syslog": false

// TLS settings for the web application
"tls": {

  // Set false to use HTTP instead. Default is true
  // "enabled": true
}
```

Currently there is not proper TLS settings/verification for CA or PEM files. This will be present in the future.

## Data Store

Fusty needs to have a location to store the backup information. This is configured in the `data_store` section. The type
of data store is specified via `type`. Currently the only supported type is `git`. Below is an example JSON
configuration with comments explaining each part.

```json
"data_store": {

  // Git is the only supported type
  "type": "git",

  // All git settings must go under the "git" section
  "git": {

    // The repository path. See https://git-scm.com/docs/git-clone#URLS
    "url": "http://myserver.local/my/repository.git",

    // If present, this will use a specific sub directory under the git repository to store the results
    // "directory": "/somesubdirectory"
    "user": {

      // The required git user.name value that will be used when committing
      "friendly_name": "John Doe",

      // The required git user.email value that will be used when committing
      "email": "johndoe@myserver.local",

      // The credentials to authenticate with. SSH authentication not yet supported
      "name": "johndoe",
      "pass": "johndoepass"
    }

    // The number of copies of the repository to maintain locally. Default is 20.
    // "pool_size": 20

    // The structure to store the backups in. Default is by_device.
    // "structure": ["by_device"]

    // Include overviews in README.md file at the top of every directory. Default is true.
    // "include_readme_overviews": true
  }
}
```

For more information about the settings and using the git data store in general, see the [Data Store](data.md)
documentation.

## Job Store

A job store is where the configuration information for jobs is stored and retrieved from. Currently, Fusty only supports
local job stores currently. Below is an example of a local job store configuration in JSON with comments explaining each
part.

```json
"job_store": {

  // Local is the only supported type
  "type" = "local",

  // All local job configs must go under the "local" section
  "local": {

    // Generics are essentially "templates" that can be applied to multiple/all jobs
    "job_generics": {

      // The "default" generic is applied to all jobs that don't specify their own generic
      "default": {

        // Prompt is just one of many settings that can be set per job. They are not all listed here
        "prompt": {
          "ends_with": "#"
        }
      },

      // This is an example of a specific generic
      "caret_prompt": {
        "prompt": {
          "ends_with": ">"
        }
      }
    },

    // All jobs are listed here
    "jobs": {

      // This is the name of the job that will be present in the data store and is referenced from the device
      "cisco_show_run" = {

        // The generic settings to inherit. Default is "default"
        // "generic": "default"

        // A required schedule indicating the frequency of execution
        "schedule": {

          // Cron is one of the three supported schedule formats, see jobs documentation for more
          "cron": "0,30 * * * *"
        },

        // The command(s) to execute and get results for
        "command": {

          // Inline is one option to provide a set of commands
          "inline": ["show run"]
        }
      }
    }
  }
}
```

There are many settings a job can have. Please reference the [Jobs](jobs.md) documentation for more information.

## Device Store

A device store is where device lists and their access information is stored. Currently only the local device store is
supported. Below is an example of device configuration in JSON with comments.

```json
"device_store": {

  // Local is the only supported type
  "type" = "local",

  // All local device configs must go under the "local" section
  "local": {

    // Generics are essentially "templates" that can be applied to multiple/all devices
    "device_generics": {

      // The "default" generic is applied to all devices that don't specify their own generic
      "default": {

        // Protocol is just one of many settings that can be set per device. They are not all listed here
        // "protocol": "ssh"
      },

    // All devices are listed here
    "devices": {

      // This is the IP or hostname of the device that is used during connection
      "device1.local": {

        // The generic settings to inherit. Default is "default"
        // "generic": "default"

        // Tags can be supplied per device. This helps a worker choose what work to do
        "tags": ["dallas-dmz-1"],

        // These are device credentials. They are among many settings for devices...
        "credentials": {
          "user": "myuser",
          "pass": "mypass"
        },

        // All jobs, by their name
        "jobs": {

          // The key is the name of the job. If there are no specific settings needed by the job,
          // it should be an empty object
          "cisco_show_run": {}
        }
      }
    }
  }
}
```

There are many settings a device can have. Please reference the [Devices](devices.md) documentation for more
information.