[![Code Climate](https://codeclimate.com/github/xshellinc/iotit/badges/gpa.svg)](https://codeclimate.com/github/xshellinc/iotit)
[![Test Coverage](https://codeclimate.com/github/xshellinc/iotit/badges/coverage.svg)](https://codeclimate.com/github/xshellinc/iotit/coverage)


IOTIT command line flashing utility
==========================


**iotit** (written in Golang) is a command line utility for flashing Single Board Computers (SBCs, aka IoT devices).


INSTALLATION
------------

### Install all requirements

```
go get ./...
```


### DEVELOPMENT ENVIROMENT

To build and run with debug log use:

```
./build.sh && ./iotit
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

REQUIREMENTS
------------

IotIT requires [VirtualBox](https://www.virtualbox.org/) with correlating version of [Extension Pack](https://www.virtualbox.org/wiki/Downloads) to be installed.


