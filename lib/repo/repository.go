package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/xshellinc/iotit/lib"
	"github.com/xshellinc/tools/constants"
	"github.com/xshellinc/tools/lib/help"
	"gopkg.in/cheggaaa/pb.v1"
)

const S3Bucket = "https://s3-ap-northeast-1.amazonaws.com/isaax-distro/versions.json"

var baseDir = filepath.Join(help.UserHomeDir(), ".isaax")
var imageDir = filepath.Join(baseDir, "images")
var vboxDir = filepath.Join(baseDir, "virtualbox")

func init() {
	help.CreateDir(baseDir)
	help.CreateDir(imageDir)
	help.CreateDir(vboxDir)
}

type Repository interface {
	//version of latest distro
	GetVersion() string
	//url of distro
	GetURL() string
	//name of the latest distro file
	Name() string
	//base dir of repository
	Dir() string
}

type (
	Raspberry struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	Edison struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	Beaglebone struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	Nanopi struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	Chirimen struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	GenericRepository struct {
		Version   string
		URL       string
		Directory string
	}
	VMs struct {
		Sd     *VMSd     `json:"vm-sd"`
		Edison *VMEdison `json:"vm-edison"`
	}
	VMSd struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	VMEdison struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
)

type S3Repository struct {
	Raspberry  `json:"raspberry"`
	Edison     `json:"edison"`
	Beaglebone `json:"beaglebone"`
	Nanopi     `json:"nanopi"`
	Chirimen   `json:"chirimen"`
}

type S3RepositoryVM struct {
	VMs `json:"vms"`
}

/*
	Raspberry Repository
*/

func (r *Raspberry) GetVersion() string {
	return r.Version
}

func (r *Raspberry) GetURL() string {
	return r.URL
}

func (r *Raspberry) Name() string {
	tokens := strings.Split(r.URL, "/")
	return tokens[len(tokens)-1]
}

func (r *Raspberry) Dir() string {
	rasp_repo := filepath.Join(imageDir, "raspberry")
	help.CreateDir(rasp_repo)
	return rasp_repo
}

/*
	Edison Repository
*/

func (r *Edison) GetVersion() string {
	return r.Version
}

func (r *Edison) GetURL() string {
	return r.URL
}

func (r *Edison) Name() string {
	tokens := strings.Split(r.URL, "/")
	return tokens[len(tokens)-1]
}

func (r *Edison) Dir() string {
	edison_repo := filepath.Join(imageDir, "edison")
	help.CreateDir(edison_repo)
	return edison_repo
}

/*
	NanoPi Repository
*/
func (n *Nanopi) GetVersion() string {
	return n.Version
}

func (n *Nanopi) GetURL() string {
	return n.URL
}

func (n *Nanopi) Dir() string {
	rasp_repo := filepath.Join(imageDir, "nanopi")
	help.CreateDir(rasp_repo)
	return rasp_repo
}

func (n *Nanopi) Name() string {
	tokens := strings.Split(n.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Beaglebone Repository
*/
func (n *Beaglebone) GetVersion() string {
	return n.Version
}

func (n *Beaglebone) GetURL() string {
	return n.URL
}

func (n *Beaglebone) Dir() string {
	beagle_repo := filepath.Join(imageDir, "beaglebone")
	help.CreateDir(beagle_repo)
	return beagle_repo
}

func (n *Beaglebone) Name() string {
	tokens := strings.Split(n.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Chirimen Repository
*/
func (c *Chirimen) GetVersion() string {
	return c.Version
}

func (c *Chirimen) GetURL() string {
	return c.URL
}
func (n *Chirimen) Dir() string {
	nano_pi_repo := filepath.Join(imageDir, "chirimen")
	help.CreateDir(nano_pi_repo)
	return nano_pi_repo
}

func (n *Chirimen) Name() string {
	tokens := strings.Split(n.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Generic Repository
*/

func (g *GenericRepository) GetVersion() string {
	return g.Version
}

func (g *GenericRepository) GetURL() string {
	return g.URL
}

func (g *GenericRepository) Dir() string {
	return g.Directory
}

func (g *GenericRepository) Name() string {
	tokens := strings.Split(g.URL, "/")
	return tokens[len(tokens)-1]
}

func (v *VMSd) GetVersion() string {
	return v.Version
}

func (v *VMSd) GetURL() string {
	return v.URL
}

func (*VMSd) Dir() string {
	sd_repo := filepath.Join(vboxDir, "sd")
	return sd_repo
}

func (v *VMSd) Name() string {
	tokens := strings.Split(v.URL, "/")
	return tokens[len(tokens)-1]
}

func (v *VMEdison) GetVersion() string {
	return v.Version
}

func (v *VMEdison) GetURL() string {
	return v.URL
}

func (*VMEdison) Dir() string {
	edison_repo := filepath.Join(vboxDir, "edison")
	return edison_repo
}

func (v *VMEdison) Name() string {
	tokens := strings.Split(v.URL, "/")
	return tokens[len(tokens)-1]
}

func NewRepository(deviceType string) (Repository, error) {
	var (
		client http.Client
		url    = S3Bucket
		repo   S3Repository
	)
	//@todo checking via timeout
	resp, err := client.Get(url)
	if err != nil {
		log.Error("Could not make GET request to url:", url, " error msg:", err.Error())
		fmt.Println("[-] Could not connect to S3 bucket")
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&repo)
	if err != nil {
		log.Error("Could not unmarshall json struct ", "error msg:", err.Error())
		return nil, err
	}
	switch deviceType {
	case constants.DEVICE_TYPE_RASPBERRY:
		return &repo.Raspberry, nil
	case constants.DEVICE_TYPE_EDISON:
		return &repo.Edison, nil
	case constants.DEVICE_TYPE_NANOPI:
		return &repo.Nanopi, nil
	case constants.DEVICE_TYPE_BEAGLEBONE:
		return &repo.Beaglebone, nil
	default:
		return nil, errors.New("unknown device type")
	}

}

func NewRepositoryVM(vmType string) (Repository, error) {
	var (
		client http.Client
		url    = S3Bucket
		repo   S3RepositoryVM
	)
	resp, err := client.Get(url)
	if err != nil {
		log.Error("Could not make GET request to url:", url, " error msg:", err.Error())
		fmt.Println("[-] Could not connect to S3 bucket")
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&repo)
	if err != nil {
		log.Error("Could not unmarshall json struct ", "error msg:", err.Error())
		return nil, err
	}
	switch vmType {
	case lib.VBOX_TEMPLATE_SD:
		return repo.Sd, nil
	case lib.VBOX_TEMPLATE_EDISON:
		return repo.Edison, nil
	default:
		return nil, errors.New("unknown virtual machine type")
	}

}

func DownloadAsync(repo Repository, wg *sync.WaitGroup) (string, *pb.ProgressBar, error) {
	dst := filepath.Join(repo.Dir(), repo.GetVersion())
	return help.DownloadFromUrlWithAttemptsAsync(repo.GetURL(), dst, 3, wg)
}

func NewGenericRepository(url, version string, dir string) Repository {
	help.CreateDir(dir)
	return &GenericRepository{
		URL:       url,
		Version:   version,
		Directory: dir,
	}
}

func VirtualBoxRepository() Repository {
	rp, err := NewRepositoryVM(lib.VBOX_TEMPLATE_SD)
	if err != nil {
		fmt.Println("[-] Could not fetch remote version")
		return nil
	}
	//return NewGenericRepository("https://s3-ap-northeast-1.amazonaws.com/isaax-distro/vm/sd/0.1.0/isaax-box-sd.zip", "0.0.1", "virtualbox/sd/")
	return NewGenericRepository(rp.GetURL(), rp.GetVersion(), rp.Dir())

}

func VirtualBoxRepositoryEdison() Repository {
	rp, err := NewRepositoryVM(lib.VBOX_TEMPLATE_EDISON)
	if err != nil {
		fmt.Println("[-] Could not fetch remote version")
		return nil
	}
	//return NewGenericRepository("https://s3-ap-northeast-1.amazonaws.com/isaax-distro/vm/sd/0.1.0/isaax-box-sd.zip", "0.0.1", "virtualbox/sd/")
	return NewGenericRepository(rp.GetURL(), rp.GetVersion(), rp.Dir())
}
