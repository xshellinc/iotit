package virtualbox

// NIC represents a virtualized network interface card.
type NIC struct {
	Network         NICNetwork
	Hardware        NICHardware
	HostonlyAdapter string
}

// NICNetwork represents the type of NIC networks.
type NICNetwork string

const (
	// NICNetAbsent sets NIC to "none"
	NICNetAbsent = NICNetwork("none")
	// NICNetDisconnected sets NIC to "null"
	NICNetDisconnected = NICNetwork("null")
	// NICNetNAT sets NIC to "nat"
	NICNetNAT = NICNetwork("nat")
	// NICNetBridged sets NIC to "bridged"
	NICNetBridged = NICNetwork("bridged")
	// NICNetInternal sets NIC to "intnet"
	NICNetInternal = NICNetwork("intnet")
	// NICNetHostonly sets NIC to "hostonly"
	NICNetHostonly = NICNetwork("hostonly")
	// NICNetGeneric sets NIC to "generic"
	NICNetGeneric = NICNetwork("generic")
)

// NICHardware represents the type of NIC hardware.
type NICHardware string

const (
	// AMDPCNetPCIII sets Am79C970A as virtualized NIC HW
	AMDPCNetPCIII = NICHardware("Am79C970A")
	//AMDPCNetFASTIII sets Am79C973 as virtualized NIC HW
	AMDPCNetFASTIII = NICHardware("Am79C973")
	//IntelPro1000MTDesktop sets 82540EM as virtualized NIC HW
	IntelPro1000MTDesktop = NICHardware("82540EM")
	//IntelPro1000TServer sets 82543GC as virtualized NIC HW
	IntelPro1000TServer = NICHardware("82543GC")
	//IntelPro1000MTServer sets 82545EM as virtualized NIC HW
	IntelPro1000MTServer = NICHardware("82545EM")
	//VirtIO sets virtio NIC
	VirtIO = NICHardware("virtio")
)
