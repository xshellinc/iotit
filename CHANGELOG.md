## [0.2.0]

### [FIXED]
- Fix Edison static IP
- Fix Edison flash
- Fix Edison password prompt

### [ADDED]
- `ssh-copy-id` SBC to disable too many password prompts
- Windows support

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
