# Architecture

## Controller and Worker

Fusty operates in a simple controller-worker fashion. A controller knows about all devices that need to be polled and
workers ask for work to do. Unlike many systems that have workers to alleviate load, the main purpose for workers in
Fusty is to help traverse possible network limitations. A controller can also be a worker in a small setup.

Workers are currently stateless which means they request work at a regular interval and the controller is expected to
give the work out. In the future workers may become more autonomous and execute schedules even when they cannot reach a
controller. Currently the command execution may be off by some time due to no worker asking for work.

## Scalability and High Availability

Controllers are not currently scalable or highly available. Only a single server is supported as a controller currently.
This means if a scheduled job was supposed to run and the controller is down, it will not run until its next scheduled
time. In the future, multi-controller models may be supported.

Since workers are currently stateless, they are theoretically infinitely scalable.