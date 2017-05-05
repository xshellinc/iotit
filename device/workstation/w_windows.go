package workstation

// @todo add windows methods

type windows struct {
	*workstation
}

func newWorkstation() WorkStation {
	return &windows{}
}
