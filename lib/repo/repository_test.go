package repo

import (
	"strings"
	"sync"
	"testing"
)

var repo Repository

func setUpRaspberryRepository() {
	repo, _ = NewRepository("raspberry-pi")
}

func TestRepository_NewRepository(t *testing.T) {
	repo, err := NewRepository("raspberry-pi")
	if err != nil {
		t.Error(err)
	}
	if repo.GetVersion() == "" {
		t.Errorf("Expecting %s got %s", "2.0.0", repo.GetVersion())
	}
}

func TestRaspberry_GetURL(t *testing.T) {
	setUpRaspberryRepository()
	if repo.GetURL() == "" {
		t.Errorf("Expecting non-empty url. Got %s ", repo.GetURL())
	}
}

func TestDownload(t *testing.T) {
	setUpRaspberryRepository()
	var wg sync.WaitGroup
	filename, _, err := DownloadAsync(repo, &wg)
	if err != nil {
		t.Error(err)
	}
	if strings.EqualFold(filename, "NOOBS_v"+repo.GetVersion()+".zip") {
		t.Errorf("Expected file name %s got %s", "NOOBS_v"+
			strings.Replace(repo.GetVersion(), ".", "_", -1)+".zip", filename)
	}
}
