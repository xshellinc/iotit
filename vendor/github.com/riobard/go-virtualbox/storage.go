package virtualbox

// StorageController represents a virtualized storage controller.
type StorageController struct {
	SysBus      SystemBus
	Ports       uint // SATA port count 1--30
	Chipset     StorageControllerChipset
	HostIOCache bool
	Bootable    bool
}

// SystemBus represents the system bus of a storage controller.
type SystemBus string

const (
	// SysBusIDE represents ide bus
	SysBusIDE = SystemBus("ide")
	// SysBusSATA represents sata bus
	SysBusSATA = SystemBus("sata")
	// SysBusSCSI represents scsi bus
	SysBusSCSI = SystemBus("scsi")
	// SysBusFloppy represents floppy bus
	SysBusFloppy = SystemBus("floppy")
)

// StorageControllerChipset represents the hardware of a storage controller.
type StorageControllerChipset string

const (
	// CtrlLSILogic represents LSILogic storage controller
	CtrlLSILogic = StorageControllerChipset("LSILogic")
	// CtrlLSILogicSAS represents LSILogic SAS storage controller
	CtrlLSILogicSAS = StorageControllerChipset("LSILogicSAS")
	// CtrlBusLogic represents BusLogic storage controller
	CtrlBusLogic = StorageControllerChipset("BusLogic")
	// CtrlIntelAHCI represents IntelAHCI storage controller
	CtrlIntelAHCI = StorageControllerChipset("IntelAHCI")
	// CtrlPIIX3 represents PIIX3 storage controller
	CtrlPIIX3 = StorageControllerChipset("PIIX3")
	// CtrlPIIX4 represents PIIX4 storage controller
	CtrlPIIX4 = StorageControllerChipset("PIIX4")
	// CtrlICH6 represents ICH6 storage controller
	CtrlICH6 = StorageControllerChipset("ICH6")
	// CtrlI82078 represents I82078 storage controller
	CtrlI82078 = StorageControllerChipset("I82078")
)

// StorageMedium represents the storage medium attached to a storage controller.
type StorageMedium struct {
	Port      uint
	Device    uint
	DriveType DriveType
	Medium    string // none|emptydrive|<uuid>|<filename|host:<drive>|iscsi
}

// DriveType represents the hardware type of a drive.
type DriveType string

const (
	// DriveDVD represents DVD drive
	DriveDVD = DriveType("dvddrive")
	// DriveHDD represents HDD drive
	DriveHDD = DriveType("hdd")
	// DriveFDD represents floppy drive
	DriveFDD = DriveType("fdd")
)
