IoTit SBC flashing tool
==========================


**IoTit** (written in Golang) is an open source command-line utility for flashing (initializing) Linux powered IoT devices.

`iotit` contains a VirtualBox wrapper [go-virtualbox](https://github.com/riobard/go-virtualbox), so it can run on OS that allows installation of VirtualBox.

SUPPORTED DEVICES
-----------

* [NanoPi NEO](http://nanopi.io/nanopi-neo.html)
* [Raspberry Pi](https://www.raspberrypi.org/)
* [Intel Edison](https://software.intel.com/en-us/iot/hardware/edison)
* [BeagleBone](http://beagleboard.org/bone)


REQUIREMENTS
------------
golang >= 1.8

INSTALLATION
------------

The easiest way is to go to [Isaax - binary distribution page](https://isaax.io/downloads/) and download a precompiled binary that matches your OS.

*Note:* `iotit` requires [VM VirtualBox](http://www.oracle.com/technetwork/server-storage/virtualbox/downloads/index.html) and [Extension Pack](http://www.oracle.com/technetwork/server-storage/virtualbox/downloads/index.html#extpack) to be installed on your machine.


If you want to build binaries yourself, then follow the regular recommendations for [go build](https://golang.org/pkg/go/build/)

*Note:* Install all requirements before trying to build it on your local workstation:

```
go get ./...
```


### DEVELOPMENT ENVIRONMENT

To build and run with debug log use:

```
./build.sh && ./iotit
```

COMMANDS
--------

To see available commands launch `iotit -h`

```
NAME:
   iotit - Flashing Tool for iot devices used by Isaax Cloud

USAGE:
   iotit [global options] [commands]

   options and commands are not mandatory

COMMANDS:
   gl, global         install to global app environment
   un, uninstall      uninstall this app
   update             update binary and vbox images
   v, version         display current version
   h, help            display help

GLOBAL OPTIONS:
   -dev [device-type]  executes iotit with specified deviceType
   -help, -h           show help
   -version, -v        print the version
```

VIRTUALBOX
--------
Virtualbox uses 2 images
 - iotit-box-sd - used during flashing nano-pi, raspberry-pi and beaglebone boards
 - iotit-box-edison - used during flashing Edison board

During installation user can choose `default` virtualbox specs

Alternatively user can create their own vbox spec by choosing `Create new virtual machine`.
This will create a spec file with a name of virtualbox and specs such as memory, cpu, vram etc,
which is applied to `iotit-box-sd` or `iotit-box-edison`



概要
----------------

IoTitを使うことによって、RaspberryPi、Intel Edison、Beaglebone、NanoPi のようなL
inux系のシングルボードコンピュータを簡単に初期化することができます。

IoTitはOpen SourceのSingle Board Computer向けフラッシュツールです。
これを使うことでより簡単にSingle Board Computerをセットアップできます。

IoTitは内部でVirtual Boxを使っておりVBのAPIを使うことで、自分専用のカスタマイズも可能です。
現在は"NanoPI Neo"や"Raspberry PI"、"Intel Edison" "BeagleBone"の4つに対応しています。
インストールや使い方はシンプルなので上記の"INSTALLATION"を読んで使ってみてください。
