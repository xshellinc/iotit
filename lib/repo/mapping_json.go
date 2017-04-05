package repo

const (
	file = "mapper.json"

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
		"Url":"http://director.downloads.raspberrypi.org/raspbian_lite/images/raspbian_lite-2017-03-03/2017-03-02-raspbian-jessie-lite.zip",
		"Title":"Raspbian jessie light"
	      },
	      {
		"Url":"http://director.downloads.raspberrypi.org/raspbian/images/raspbian-2017-03-03/2017-03-02-raspbian-jessie.zip",
		"Title":"Raspbian jessie w Pixel"
	      }
	    ]
	  },
	  {
	    "Name":"edison",
	    "Images":[
	      {
		"Url":"http://iotdk.intel.com/images/3.5/edison/iot-devkit-prof-dev-image-edison-20160606.zip"
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
		    "Url":"http://www.mediafire.com/file/izyvj97my2h0qbj/nanopi-m1-plus-ubuntu-core-qte-sd4g-20170227.img.zip",
		    "Title":"Light Ubuntu-core"
		  },
		  {
		    "Url":"http://www.mediafire.com/file/68m7e1q4vsr3hlc/nanopi-m1-plus-debian-sd4g-20170228.img.zip",
		    "Title":"Debian"
		  }
		]
	      },
	      {
		"Name":"NanoPi M1",
		"Images":[
		  {
		    "Url":"http://www.mediafire.com/file/kfihz987rf6s87d/nanopi-m1-debian-sd4g-20170204.img.zip",
		    "Title":"Light Ubuntu-core"
		  },
		  {
		    "Url":"http://www.mediafire.com/file/kfihz987rf6s87d/nanopi-m1-debian-sd4g-20170204.img.zip",
		    "Title":"Debian"
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
		    "Url":"http://www.mediafire.com/file/dts72ru5vzzzsr5/s5p6818-ubuntu-core-qte-sd4g-20170316.img.zip",
		    "Title":"Light Ubuntu-core"
		  },
		  {
		    "Url":"http://www.mediafire.com/file/bb5xf8t4203m89y/s5p6818-debian-sd4g-20170316.img.zip",
		    "Title":"Debian"
		  },
		  {
		    "Url":"http://www.mediafire.com/file/e19c3nhjvm56ico/s5p6818-debian-wifiap-sd4g-20170316.img.zip",
		    "Title":"Debian wifiap"
		  }
		]
	      },
	      {
		"Name":"NanoPi NEO",
		"Images":[
		  {
		    "Url":"http://www.mediafire.com/file/5524t880fht9vtq/nanopi-neo-ubuntu-core-qte-sd4g-20170331.img.zip"
		  }
		]
	      },
	      {
		"Name":"NanoPi NEO Air",
		"Images":[
		  {
		    "Url":"http://www.mediafire.com/file/h6y011436mc3qs3/nanopi-air-ubuntu-core-qte-sd4g-20170220.img.zip"
		  }
		]
	      },
	      {
		"Name":"NanoPi S2",
		"Images":[
		  {
		    "Url":"http://www.mediafire.com/file/h6y011436mc3qs3/nanopi-air-ubuntu-core-qte-sd4g-20170220.img.zip",
		    "Title":"Light Ubuntu-core"
		  }
		]
	      },
	      {
		"Name":"NanoPi a64",
		"Images":[
		  {
		    "Url":"http://www.mediafire.com/file/hzfhcu0r1bb9ogt/nanopi-a64-core-qte-sd4g-20161129.img.zip",
		    "Title":"Light Ubuntu-core"
		  },
		  {
		    "Url":"http://www.mediafire.com/file/cbh9mkb70p18m12/nanopi-a64-ubuntu-mate-sd4g-20161129.img.zip",
		    "Title":"Ubuntu with a MATE-desktop"
		  }
		]
	      }
	    ],
	    "Images":[
	      {
		"Url":"http://www.mediafire.com/file/9ooty82tkzh88bb/s5p4418-ubuntu-core-qte-sd4g-20170316.img.zip",
		"Title":"Light Ubuntu-core"
	      },
	      {
		"Url":"http://www.mediafire.com/file/eykqbtm7en6dwzy/s5p4418-debian-sd4g-20170316.img.zip",
		"Title":"Debian"
	      },
	      {
		"Url":"http://www.mediafire.com/file/4g7uniiuvtmlfva/s5p4418-android-sd4g-20170307.img.zip",
		"Title":"Debian wifiap"
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
		    "Url":"https://debian.beagleboard.org/images/bbx15-debian-8.6-lxqt-4gb-armhf-2016-11-06-4gb.img.xz"
		  }
		]
	      },
	      {
		"Name":"BeagleBoard-xM",
		"Images":[
		  {
		    "Url":"https://debian.beagleboard.org/images/bbxm-debian-8.6-lxqt-xm-4gb-armhf-2016-11-06-4gb.img.xz"
		  }
		]
	      }
	    ],
	    "Images":[
	      {
	      	"Url":"https://rcn-ee.com/rootfs/2017-03-09/elinux/ubuntu-16.04.2-console-armhf-2017-03-09.tar.xz",
	      	"Title":"Ubuntu minimal"
	      },
	      {
		"Url":"https://debian.beagleboard.org/images/bone-debian-8.7-lxqt-4gb-armhf-2017-03-19-4gb.img.xz",
		"Title":"Debian with desktop"
	      }
	    ]
	  }
	]
}
`
)
