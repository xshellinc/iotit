ISAAX iotit flashing tool
==========================

VERSION 0.1.0

LAST UPDATE 2017-02-16

IotIT (written in Golang) is a Flashing Tool for iot devices used by Isaax Cloud


OVER VIEW
-----------
IoTit it an open source Flashing Tool for Single Board Computers.
IoTit using "Virtual Box" for It's Flashing ,thus You can also using VB API for customaization.
Currently supoorting "NanoPi Neo" and "Raspberry Pi" "intel edison" "Beagle Bone"



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


ISAAX iotit flashing tool
==========================

バージョン0.1.0

最終アップデート2017年２月１６日

IoTitはGolangで書かれたフラッシュツールでisaax cloudで使われていたシステムを分離独立させたものです。


概要
----------------
IoTitはOpen SourceのSingle Board Computer向けフラッシュツールです。
これを使うことでより簡単にSingle Board Computerをセットアップできます。
IoTitは内部でVirtual Boxを使っておりVBのAPIを使うことで、自分専用のカスタマイズも可能です。
現在は"NanoPI Neo"や"Raspberry PI"、"intel edison" "Beagle Bone"の4つに対応しています。
インストールや使い方はシンプルなので上記の"INSTALLATION"を読んで使ってみてください。
