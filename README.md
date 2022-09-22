# README

## Overview

The Core is the module responsible for managing resources and instances. It embeds an image server that handles media and disk images. 

All functions of Nano provide as REST API of the Core, any command or request need process should submit to the Core. 

It is also the stub of network discovery, so remember to start the Core before any other modules.



Binary release found [here](<https://github.com/project-nano/releases/releases>)

See more detail for [Quick Guide](<https://nanocloud.readthedocs.io/projects/guide/en/latest/concept.html>)

Official Site: <https://nanos.cloud/en-us/>

REST API: <https://nanoen.docs.apiary.io/>

Wiki: <https://github.com/project-nano/releases/wiki/English>

## Build

Assume that the golang lib installed in the '/home/develop/go',  and source code downloaded in the path '/home/develop/nano/core'.

Set environment variable GOPATH before compiling

```
#git clone https://github.com/project-nano/core.git
#cd core
#go build -o core -ldflags="-w -s"
```



## Command Line

All Nano modules provide the command-line interface, and called like :

< module name > [start | stop | status | halt]

- start: start module service, output error message when start failed, or version information.
- stop: stops the service gracefully. Releases allocated resources and notify any related modules.
- status: checks if the module is running.
- halt: terminate service immediately.

You can call the Core module both in the absolute path and relative path.

```
#cd /opt/nano/core
#./core start

or

#/opt/nano/core/core start
```

Please check the log file "log/core.log" when encountering errors.

## Configure

All configure files stores in the path: config

### Domain Configuration

Filename: domain.cfg

See more detail about [Domain](<https://nanocloud.readthedocs.io/projects/guide/en/latest/concept.html#communicate-domain>)

| Parameter          | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| **domain**         | The name of a communication domain, like 'nano' in default, only allows characters. |
| **group_port**     | Multicast port, 5599 in default                              |
| **group_address**  | Multicast address, '224.0.0.226' in default.                 |
| **listen_address** | Listening Address of the core service，string in the IPv4 format |

### API Configuration

Filename: api.cfg

| Parameter | Description                                 |
| --------- | ------------------------------------------- |
| **port**  | listening port of REST API, 5850 in default |



### Image Server Configuration

Filename：image.cfg

| Parameter     | Description                                                  |
| ------------- | ------------------------------------------------------------ |
| **cert_file** | the file-path of TLS certificate used in image transportation |
| **key_file**  | the file-path of TLS key used in image transportation        |

