## [0.3.3]

### [ADDED]
- Flashing Toradex Colibri iMX6 support

### [UPDATED]
- Improved log command

### [FIXED]
- Bugfix


## [0.3.0]

### [ADDED]
- New commands added (help, log, configure, list)
- Unattended mode (with `-q` flag provided don't ask any questions)
- Flashing esp32/esp8266 support

### [UPDATED]
- Updated images for raspberry pi, nanopi and beaglebone
- Vendor dependencies

### [FIXED]
- Bugfix and refactoring


## [0.2.2]

### [FIXED]
- Windows platform support improvements (edison flashing in particular)
- Custom boards flashing

### [REMOVED]
- Removed unnecessary mounting point selection dialog

### [UPDATED]
- Update NanoPi images links


## [0.2.1]

### [FIXED]
- Fix ssh enabling for Raspberry Pi

### [ADDED]
- Add custom image flashing


## [0.2.0]

### [FIXED]
- Fix Edison static IP
- Fix Edison flash
- Fix Edison password prompt

### [ADDED]
- `ssh-copy-id` SBC to disable too many password prompts
- Experimental windows platform support
- Ability to enable SSH on raspberry pi

### [REMOVED]
- VBox VM check that caused `iotit` to crash
- 2nd prompt to stop VM

### [CHANGED]
- Edison WiFi config step before setting static IP


## Changed 0.1.1

- Fix text typos
- Fix Beaglebone configuration failures
- Fix static IP failures
- Fix forever loop in case of flashing error
- Fix wrong repo file for a nanopi board

- Improve edison interfaces filtration when configuring

- Add global binary installation and un-installation
- Add Codeclimate badges
- Add dialog to append 8.8.8.8 as a secondary dns
- Add Current configuration name in vbox description

- Change repo address to cdn
- Code refactoring
