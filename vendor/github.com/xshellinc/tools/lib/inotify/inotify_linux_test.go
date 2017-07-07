package inotify

import (
	"fmt"
	"os"
	"testing"
)

func TestInotify(t *testing.T) {
	var name = "/tmp/test_inotify"

	f, err := os.Create(name)
	if err != nil {
		t.Error(err)
	}

	in, err := New()
	if err != nil {
		t.Error(err)
	}

	_, err = in.Add(name, InModify)
	if err != nil {
		t.Error(err)
	}

	go func() {
		for i := 0; i < 10; i++ {
			f.WriteString("test")
			f.Sync()
		}
	}()

	for i := 0; i < 10; i++ {
		ev := <-in.C
		fmt.Println(ev)
	}

	if err := in.Close(); err != nil {
		t.Error(err)
	}
}
