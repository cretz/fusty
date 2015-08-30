# API

## Authentication

TODO (none for now)

## Calls

### GET /worker/next?tag=tag1&tag=tag2&seconds=N&max=M

Exclusively obtain the next set of jobs for the next N seconds guaranteeing that no more than M jobs are returned. If no
tags are provided, all tags are assumed. If no second count is provided, 15 is assumed. If no max is provided, 15 is
assumed.

Success is 200 or 204, anything else is failure. If result is 204, there are no jobs to process. If response is 200, the
body is a JSON array of individual jobs. This basically includes a [job](jobs.md) and a [device](devices.md) with every
value expanded as necessary and no generics. Example response:

```js
[
  {
    "device": {
      "name": "device1.local",
      "host": "device1.local",
      "protocol": {
        "type": "ssh",
        "ssh": {
          "port": 22
        }
      },
      "tags": ["dallas-dmz-1"],
      "credentials": {
        "user": "myuser",
        "pass": "mypass",
        "prompt": {
          "ends_with": "#"
        }
      }
    },
    "job": {
      "name": "cisco_show_run",
      "schedule": {
        "cron": "0,30 * * * *"
      },
      "command": {
        "inline": ["show run"]
      },
      "prompt": {
        "ends_with": "#"
      }
    }
  },
  "timestamp": 446536800
]
```

The schedule is always a fixed unix timestamp. There are cases where a timestamp may be in the past because no worker
has asked for that job. Those should be run immediately.

### POST /worker/complete

A job completion. This is posted as multipart form fields. The form fields:

* device - The device name (not host)
* job_timestamp - The unix timestamp this was supposed to start on
* start_timestamp - The unix timestamp this actually started on
* end_timestamp - The unix timestamp this ended on
* file - The entire contents fetched post authentication, with the filename being the job name
* failure - If present, this is a simple field explaining the failure

Note, currently the entire set is held in memory. In the future streaming writes all the way to git should be supported.
Therefore, currently several large jobs (e.g. many hundreds of MB each) at the same time could overload the system.