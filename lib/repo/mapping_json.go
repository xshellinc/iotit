package repo

const (
	file = "mapping.json"

	// example is used to create mapper.json file or used in case of absence of the file
	example = `{
"Devices":
	[
	  {
	    "Name":"raspberry-pi",
	    "Sub":[
	      {
		"Name":"Raspberry Pi Model A, A+, B, B+, Zero, Zero W"
	      },
	      {
		"Name":"Raspberry Pi 2 (based on Model B+)"
	      },
	      {
		"Name":"Raspberry Pi 3"
	      }
	    ],
	    "Images":[
	      {
		"URL":"http://director.downloads.raspberrypi.org/raspbian_lite/images/raspbian_lite-2017-03-03/2017-03-02-raspbian-jessie-lite.zip",
		"Title":"Raspbian jessie light",
		"User":"pi",
		"Pass":"raspberry"
	      },
	      {
		"URL":"http://director.downloads.raspberrypi.org/raspbian/images/raspbian-2017-03-03/2017-03-02-raspbian-jessie.zip",
		"Title":"Raspbian jessie w Pixel",
		"User":"pi",
		"Pass":"raspberry"
	      }
	    ]
	  },
	  {
	    "Name":"edison",
	    "Images":[
	      {
		"URL":"http://iotdk.intel.com/images/3.5/edison/iot-devkit-prof-dev-image-edison-20160606.zip",
		"User":"root"
	      }
	    ]
	  },
	  {
	    "Name":"nano-pi",
	    "Sub":[
	      {
		"Name":"NanoPi2"
	      },
	      {
		"Name":"NanoPi 2 Fire"
	      },
	      {
		"Name":"NanoPi M1 Plus",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/izyvj97my2h0qbj/nanopi-m1-plus-ubuntu-core-qte-sd4g-20170227.img.zip",
		    "Title":"Light Ubuntu-core",
		    "User":"root",
		    "Pass":"fa"
		  },
		  {
		    "URL":"http://www.mediafire.com/file/68m7e1q4vsr3hlc/nanopi-m1-plus-debian-sd4g-20170228.img.zip",
		    "Title":"Debian",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi M1",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/kfihz987rf6s87d/nanopi-m1-debian-sd4g-20170204.img.zip",
		    "Title":"Light Ubuntu-core",
		    "User":"root",
		    "Pass":"fa"
		  },
		  {
		    "URL":"http://www.mediafire.com/file/kfihz987rf6s87d/nanopi-m1-debian-sd4g-20170204.img.zip",
		    "Title":"Debian",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi M2"
	      },
	      {
		"Name":"NanoPi M3",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/dts72ru5vzzzsr5/s5p6818-ubuntu-core-qte-sd4g-20170316.img.zip",
		    "Title":"Light Ubuntu-core",
		    "User":"root",
		    "Pass":"fa"
		  },
		  {
		    "URL":"http://www.mediafire.com/file/bb5xf8t4203m89y/s5p6818-debian-sd4g-20170316.img.zip",
		    "Title":"Debian",
		    "User":"root",
		    "Pass":"fa"
		  },
		  {
		    "URL":"http://www.mediafire.com/file/e19c3nhjvm56ico/s5p6818-debian-wifiap-sd4g-20170316.img.zip",
		    "Title":"Debian wifiap",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi NEO",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/5524t880fht9vtq/nanopi-neo-ubuntu-core-qte-sd4g-20170331.img.zip",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi NEO Air",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/h6y011436mc3qs3/nanopi-air-ubuntu-core-qte-sd4g-20170220.img.zip",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi S2",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/h6y011436mc3qs3/nanopi-air-ubuntu-core-qte-sd4g-20170220.img.zip",
		    "Title":"Light Ubuntu-core",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      },
	      {
		"Name":"NanoPi a64",
		"Images":[
		  {
		    "URL":"http://www.mediafire.com/file/hzfhcu0r1bb9ogt/nanopi-a64-core-qte-sd4g-20161129.img.zip",
		    "Title":"Light Ubuntu-core",
		    "User":"root",
		    "Pass":"fa"
		  },
		  {
		    "URL":"http://www.mediafire.com/file/cbh9mkb70p18m12/nanopi-a64-ubuntu-mate-sd4g-20161129.img.zip",
		    "Title":"Ubuntu with a MATE-desktop",
		    "User":"root",
		    "Pass":"fa"
		  }
		]
	      }
	    ],
	    "Images":[
	      {
		"URL":"http://www.mediafire.com/file/9ooty82tkzh88bb/s5p4418-ubuntu-core-qte-sd4g-20170316.img.zip",
		"Title":"Light Ubuntu-core",
		"User":"root",
		"Pass":"fa"
	      },
	      {
		"URL":"http://www.mediafire.com/file/eykqbtm7en6dwzy/s5p4418-debian-sd4g-20170316.img.zip",
		"Title":"Debian",
		"User":"root",
		"Pass":"fa"
	      },
	      {
		"URL":"http://www.mediafire.com/file/4g7uniiuvtmlfva/s5p4418-android-sd4g-20170307.img.zip",
		"Title":"Debian wifiap",
		"User":"root",
		"Pass":"fa"
	      }
	    ]
	  },
	  {
	    "Name":"beaglebone",
	    "Sub":[
	      {
		"Name":"BeagleBone (*,*Black,*Blue,*Green,*Wireless)"
	      },
	      {
		"Name":"BeagleBoard-X15",
		"Images":[
		  {
		    "URL":"https://debian.beagleboard.org/images/bbx15-debian-8.6-lxqt-4gb-armhf-2016-11-06-4gb.img.xz",
		    "User":"ubuntu",
		    "Pass":"temppwd"
		  }
		]
	      },
	      {
		"Name":"BeagleBoard-xM",
		"Images":[
		  {
		    "URL":"https://debian.beagleboard.org/images/bbxm-debian-8.6-lxqt-xm-4gb-armhf-2016-11-06-4gb.img.xz",
		    "User":"ubuntu",
		    "Pass":"temppwd"
		  }
		]
	      }
	    ],
	    "Images":[
	      {
		"URL":"https://debian.beagleboard.org/images/bone-debian-8.7-lxqt-4gb-armhf-2017-03-19-4gb.img.xz",
		"Title":"Debian with desktop",
		"User":"ubuntu",
		"Pass":"temppwd"
	      }
	    ]
	  }
	]
}
`
)
