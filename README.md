[![Build status](https://badge.buildkite.com/a079390efcfff71aca1f6e47dcefbc3d6ec95c3b2ad4108bd2.svg)](https://buildkite.com/xshell/isaax-cli)
ISAAX iotit flashing tool
==========================

VERSION 0.1.0

LAST UPDATE 2017-02-16

IotIT (written in Golang) is a Flashing Tool for iot devices used by Isaax Cloud



INSTALLATION
------------

### Install all requirements

```
go get ./...
```


### DEVELOPMENT ENVIROMENT

To build and run with debug log use:

```
./build.sh && ./isaax
```

COMMANDS
--------
###### To see available commands launch `iotit -h`
```
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options]

GLOBAL OPTIONS:
   -update <sd|edison> update vbox and dependencies
   -update <sd|edison> update vbox and dependencies
   -dev [device-type]  executes iotit with specified deviceType
   -help, -h           show help
   -version, -v        print the version
```