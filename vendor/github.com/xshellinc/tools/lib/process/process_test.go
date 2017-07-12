package process

import (
	"os"
	"testing"
)

func TestProcesses(t *testing.T) {
	ps, err := Processes()
	if err != nil {
		t.Error(err)
	}
	if len(ps) == 0 {
		t.Error("No processes found")
	}
}

func TestFind(t *testing.T) {
	ps, err := Find(os.Getpid())
	if err != nil {
		t.Error(err)
	}
	if ps == nil {
		t.Error("Unable to find process")
	}
}

func TestFindByExecutable(t *testing.T) {
	bin, err := os.Executable()
	if err != nil {
		t.Error(err)
	}

	ps, err := FindByExecutable(bin)
	if err != nil {
		t.Error(err)
	}

	if len(ps) == 0 {
		t.Error("Unable to find process")
	}
}
