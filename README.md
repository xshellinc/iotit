[![CircleCI](https://circleci.com/gh/xshellinc/iotit.svg?style=svg)](https://circleci.com/gh/xshellinc/iotit)
# IoTit SBC flashing tool
==========================


**IoTit** (written in Golang) is an open source command-line utility for flashing (initializing) Linux powered IoT devices.

`iotit` contains a VirtualBox wrapper [go-virtualbox](https://github.com/riobard/go-virtualbox), so it can run on OS that allows installation of VirtualBox.

### SUPPORTED DEVICES
-----------

* [NanoPi NEO](http://nanopi.io/nanopi-neo.html)
* [Raspberry Pi](https://www.raspberrypi.org/)
* [Intel Edison](https://software.intel.com/en-us/iot/hardware/edison)
* [BeagleBone](http://beagleboard.org/bone)


### REQUIREMENTS
------------
golang >= 1.8
virtualbox >= 5.0

### INSTALLATION
------------

The easiest way is to go to [Isaax - binary distribution page](https://isaax.io/downloads/) and download a precompiled binary that matches your OS.

*Note:* `iotit` requires [VM VirtualBox](http://www.oracle.com/technetwork/server-storage/virtualbox/downloads/index.html) and [Extension Pack](http://www.oracle.com/technetwork/server-storage/virtualbox/downloads/index.html#extpack) to be installed on your machine.

##### Windows specific
You will need to install [git-for-windows](https://git-for-windows.github.io/)

Upon installing, you will need to add both git and Unix tools to the system PATH [example screenshot](https://raw.githubusercontent.com/xshellinc/iotit/master/static/img/git-win.png)

Windows PowerShell or CMD should be run as `Administrator` for `iotit` to be able to write on external drives.

#### Building

If you want to build binaries yourself, then follow the regular recommendations for [go build](https://golang.org/pkg/go/build/)

*Note:* Install all requirements before trying to build it on your local workstation:

```
go get ./...
```

*Note:* Although it is not required we recommend to install `ssh-copy-id` for flashing edison:

```
brew install ssh-copy-id
```

### DEVELOPMENT ENVIRONMENT

To build and run with debug log use:

```
./build.sh && ./iotit
```

### COMMANDS
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

#### VIRTUALBOX
----------------
During installation user can choose `default` virtualbox specs

Alternatively user can create their own vbox spec by choosing `Create new virtual machine`.
This will create a spec file with a name of virtualbox and specs such as memory, cpu, vram etc,
which is applied to `iotit-box`


##### INTERNALS
----------------
`$HOME/.iotit` - a directory containing iotit related files
`$HOME/.iotit/mapping.json` - a file containing different device types and urls of images to be downloaded
`$HOME/.iotit/virtualbox/{version}/iotit-box.zip` - a packed virtual box
`$HOME/.iotit/images/{device}/{image_pack}` - packed images grouped by device names

`iotit` uses x64 virtualbox in order to flash and configure devices,
because it allows to work with linux partitions and reduces installation requirements
across different OS

Currently 2 workflows are supported:

##### 1 Edison:
- copy installation files into virtualbox
- run flashall.sh - to reflash edison
- run edison_configure - to configure

##### 2 Sd:
- copy installation files into virtualbox
- mount the image partition into loop via `losetup` and `mount`
- write configuration files into the image
- write image into sd-card via `dd` or `diskutil` on macos

VirtualBox uses alpine virtualbox image with additional software installed
```
bash
libusb-dev 
xz
util-lixus
dfu-util
```

Edison device is additionally mapped to the usb ports
```
Intel Edison [0310]
Intel USB download gadget [9999]
FTDI FT232R USB UART [0600]
```

#### CUSTOM BOARDS FLASHING:
----------------
In order to flash a custom board, device info should be added into `~/.iotit/mapping.json`. This file should be created as soon as you start iotit but can also be generated witht the command `iotit --help`.


#### STRUCTURE OF `mapping.json`:
----------------

##### Example:
```
"Devices":
	[
	  {
	    "Name":"device_name_or_category",
	    "Sub":[
	      {
	        "Name":"device_name_or_sub_category",
	        "Sub":[],
	        "Images:[]
	      }
	    ],
	    "Images":[
	      {
	        "Url":"url",
            "Title":"url_title"
	      }
	    ]
	  }
	]
```

If you do not specify any images for sub categeory it will choose whatever you have specified in the global image section. If you have more than one image in any image section you will be presented with a list when flashing.

#### Structure:
```
DeviceMapping struct {
    Name string
    Sub []DeviceMapping
    []Images struct {
        Url url,
        Title string
    }
}
```

#### Algorithm:
has a tree like structure -
devices are listed using `Name` field, then devices are listed within `Sub` array and etc.

If a `Sub` device doesn't have any image, then parent's images are used instead



### 概要
----------------

IoTitを使うことによって、RaspberryPi、Intel Edison、Beaglebone、NanoPi のようなL
inux系のシングルボードコンピュータを簡単に初期化することができます。

IoTitはOpen SourceのSingle Board Computer向けフラッシュツールです。
これを使うことでより簡単にSingle Board Computerをセットアップできます。

IoTitは内部でVirtual Boxを使っておりVBのAPIを使うことで、自分専用のカスタマイズも可能です。
現在は"NanoPI Neo"や"Raspberry PI"、"Intel Edison" "BeagleBone"の4つに対応しています。
インストールや使い方はシンプルなので上記の"INSTALLATION"を読んで使ってみてください。
