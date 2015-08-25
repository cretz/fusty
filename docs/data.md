# Data Store

The only currently supported data store for Fusty backups is Git. This is configured via the
[configuration](configuration.md) file. This document covers some of the high-level features of the Git data store.

## Git

### Structure

When storing backups in the Git repository, Fusty puts the result of each job in the same file. Depending upon the
configured structure, the files may be under a folder per device and a file per job name or a folder per job name and a
file per device name. For example, if the structure is configured with both `by_job` and `by_device`, the repository
might look like this:

```
├── reporoot/
│   ├── by_job/
│   │   ├── job1_name/
│   │   │   ├── device1.local.txt
│   │   │   ├── device2.local.txt
│   │   ├── job2_name/
│   │   │   ├── device1.local.txt
│   │   │   ├── device2.local.txt
│   ├── by_device/
│   │   ├── device1.local/
│   │   │   ├── job1_name.txt
│   │   │   ├── job2_name.txt
│   │   ├── device2.local/
│   │   │   ├── job1_name.txt
│   │   │   ├── job2_name.txt
```

### Pools and Atomicness

Fusty writes (or overwrites) a file for every job execution for every device. Ideally every single write would be done
asynchronously but there can be conflicts with Git when writing to the same file without having the latest update.
Therefore each write to a specific file in Fustry is queued up for that specific file. This ensures that each file is
updated with the latest information. Since there is only one file per device and job, there is rarely a concern of
writes being stalled because they are queued.

Git works on the local filesystem by changing files, committing them, and pushing them. This cannot be done at the same
time for different operations. To handle this, Fusty has a pool of Git repositories checked out that it uses to handle
the updates. Fusty waits for a pool member (i.e. a checked out instance of the repository) to become available and
queues up things to write while it does this. Once one is available, all queued up operations are sent to git. This
means the higher the configured pool size, the more work Fusty can persist at a time and the quicker it can do so
which helps prevent data loss.

Note, in the future Fusty may use Git tricks to eliminate the need for pools altogether.

Note, in the future the requirements concerning high availability may change how Fusty queues up Git updates. See the
[architecture](architecture) documentation for more information on scaling and data loss.

### Readme Overviews

Enabled by default, readme overviews put overview information in a `README.md` file at the top of every directory and
keep it updated. Since this file can cross jobs and/or devices and Fusty must update each file atomically, it can be
contentious to update readme files. Taking the example repository structure from the structure section above, here is an
overview of what each readme file would contain:

* `reporoot/by_job/README.md` - Table showing every job, links to their readmes, and last time the job was executed on
  any device.
* `reporoot/by_job/job1_name/README.md` - Table showing job overview, every device, and the last time each was updated.
* `reporoot/by_device/README.md` - Table showing every device, links to their readmes, and last time the device had any
  job executed on it.
* `reporoot/by_device/device1.local/README.md` - Table showing device overview, every job, and the last time each was
  updated.