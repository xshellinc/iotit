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

// S3Bucket keeps default S3 bucket path
const S3Bucket = "https://cdn.isaax.io/isaax-distro/versions.json"
const IoTItRepo = "https://cdn.isaax.io/iotit/version.json"

var baseDir = filepath.Join(help.UserHomeDir(), ".iotit")
var imageDir = filepath.Join(baseDir, "images")
var vboxDir = filepath.Join(baseDir, "virtualbox")

func init() {
	help.CreateDir(baseDir)
	help.CreateDir(imageDir)
	help.CreateDir(vboxDir)
}

// Repository represents image repo
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
	// Raspberry image
	Raspberry struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// Edison image
	Edison struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// Beaglebone image
	Beaglebone struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// Nanopi image
	Nanopi struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// Chirimen image
	Chirimen struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// GenericRepository is so generic
	GenericRepository struct {
		Version   string
		URL       string
		Directory string
	}
	// VMs represents virtual machines repos
	VMs struct {
		SD     *VMSD     `json:"vm-sd"`
		Edison *VMEdison `json:"vm-edison"`
	}
	// VMSD represents VM image for SD-based platforms
	VMSD struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// VMEdison represents VM image for Edison
	VMEdison struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
)

// S3Repository is a configuration entry for all images
type S3Repository struct {
	Raspberry  `json:"raspberry"`
	Edison     `json:"edison"`
	Beaglebone `json:"beaglebone"`
	Nanopi     `json:"nanopi"`
	Chirimen   `json:"chirimen"`
}

// S3RepositoryVM is a configuration entry for all VMs
type S3RepositoryVM struct {
	VMs `json:"vms"`
}

/*
	Raspberry Repository
*/

// GetVersion of RaspberryPI image
func (r *Raspberry) GetVersion() string {
	return r.Version
}

// GetURL of RaspberryPI image
func (r *Raspberry) GetURL() string {
	return r.URL
}

// Name of RaspberryPI image
func (r *Raspberry) Name() string {
	tokens := strings.Split(r.URL, "/")
	return tokens[len(tokens)-1]
}

// Dir of RaspberryPI image
func (r *Raspberry) Dir() string {
	raspRepo := filepath.Join(imageDir, "raspberry")
	help.CreateDir(raspRepo)
	return raspRepo
}

/*
	Edison Repository
*/

// GetVersion of Edison image
func (r *Edison) GetVersion() string {
	return r.Version
}

// GetURL of Edison image
func (r *Edison) GetURL() string {
	return r.URL
}

// Name of Edison image
func (r *Edison) Name() string {
	tokens := strings.Split(r.URL, "/")
	return tokens[len(tokens)-1]
}

// Dir of Edison image
func (r *Edison) Dir() string {
	edisonRepo := filepath.Join(imageDir, "edison")
	help.CreateDir(edisonRepo)
	return edisonRepo
}

/*
	NanoPi Repository
*/

// GetVersion of NanoPI image
func (n *Nanopi) GetVersion() string {
	return n.Version
}

// GetURL of NanoPI image
func (n *Nanopi) GetURL() string {
	return n.URL
}

// Dir of NanoPI image
func (n *Nanopi) Dir() string {
	raspRepo := filepath.Join(imageDir, "nanopi")
	help.CreateDir(raspRepo)
	return raspRepo
}

// Name of NanoPI image
func (n *Nanopi) Name() string {
	tokens := strings.Split(n.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Beaglebone Repository
*/

// GetVersion of Beaglebone image
func (n *Beaglebone) GetVersion() string {
	return n.Version
}

// GetURL of Beaglebone image
func (n *Beaglebone) GetURL() string {
	return n.URL
}

// Dir of Beaglebone image
func (n *Beaglebone) Dir() string {
	beagleRepo := filepath.Join(imageDir, "beaglebone")
	help.CreateDir(beagleRepo)
	return beagleRepo
}

// Name of Beaglebone image
func (n *Beaglebone) Name() string {
	tokens := strings.Split(n.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Chirimen Repository
*/

// GetVersion of Chirimen image
func (c *Chirimen) GetVersion() string {
	return c.Version
}

// GetURL of Chirimen image
func (c *Chirimen) GetURL() string {
	return c.URL
}

// Dir of Chirimen image
func (c *Chirimen) Dir() string {
	nanoPiRepo := filepath.Join(imageDir, "chirimen")
	help.CreateDir(nanoPiRepo)
	return nanoPiRepo
}

// Name of Chirimen image
func (c *Chirimen) Name() string {
	tokens := strings.Split(c.URL, "/")
	return tokens[len(tokens)-1]
}

/*
	Generic Repository
*/

// GetVersion of generic repo
func (g *GenericRepository) GetVersion() string {
	return g.Version
}

// GetURL of generic repo
func (g *GenericRepository) GetURL() string {
	return g.URL
}

// Dir of generic repo
func (g *GenericRepository) Dir() string {
	return g.Directory
}

// Name of generic repo
func (g *GenericRepository) Name() string {
	tokens := strings.Split(g.URL, "/")
	return tokens[len(tokens)-1]
}

// GetVersion of VM for SD-based platforms
func (v *VMSD) GetVersion() string {
	return v.Version
}

// GetURL of VM for SD-based platforms
func (v *VMSD) GetURL() string {
	return v.URL
}

// Dir of VM for SD-based platforms
func (*VMSD) Dir() string {
	sdRepo := filepath.Join(vboxDir, "sd")
	return sdRepo
}

// Name of VM for SD-based platforms
func (v *VMSD) Name() string {
	tokens := strings.Split(v.URL, "/")
	return tokens[len(tokens)-1]
}

// GetVersion of Edison VM
func (v *VMEdison) GetVersion() string {
	return v.Version
}

// GetURL of Edison VM
func (v *VMEdison) GetURL() string {
	return v.URL
}

// Dir of Edison VM
func (*VMEdison) Dir() string {
	edisonRepo := filepath.Join(vboxDir, "edison")
	return edisonRepo
}

// Name of Edison VM
func (v *VMEdison) Name() string {
	tokens := strings.Split(v.URL, "/")
	return tokens[len(tokens)-1]
}

// NewRepository creates new repository for specified device type
func NewRepository(deviceType string) (Repository, error) {
	var (
		client http.Client
		url    = S3Bucket
		repo   S3Repository
	)
	//@todo re-try if timeout
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

// NewRepositoryVM creates new repository for specified VM type
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
	case lib.VBoxTemplateSD:
		return repo.SD, nil
	case lib.VBoxTemplateEdison:
		return repo.Edison, nil
	default:
		return nil, errors.New("unknown virtual machine type")
	}

}

// DownloadAsync starts async download
func DownloadAsync(repo Repository, wg *sync.WaitGroup) (string, *pb.ProgressBar, error) {
	dst := filepath.Join(repo.Dir(), repo.GetVersion())
	return help.DownloadFromUrlWithAttemptsAsync(repo.GetURL(), dst, 3, wg)
}

// NewGenericRepository creates new generic repo
func NewGenericRepository(url, version string, dir string) Repository {
	help.CreateDir(dir)
	return &GenericRepository{
		URL:       url,
		Version:   version,
		Directory: dir,
	}
}

// VirtualBoxRepository gets currents repo status for SD platforms
func VirtualBoxRepository() Repository {
	rp, err := NewRepositoryVM(lib.VBoxTemplateSD)
	if err != nil {
		fmt.Println("[-] Could not fetch remote version")
		return nil
	}
	//return NewGenericRepository("https://s3-ap-northeast-1.amazonaws.com/isaax-distro/vm/sd/0.1.0/isaax-box-sd.zip", "0.0.1", "virtualbox/sd/")
	return NewGenericRepository(rp.GetURL(), rp.GetVersion(), rp.Dir())

}

// VirtualBoxRepositoryEdison gets currents repo status for Edison
func VirtualBoxRepositoryEdison() Repository {
	rp, err := NewRepositoryVM(lib.VBoxTemplateEdison)
	if err != nil {
		fmt.Println("[-] Could not fetch remote version")
		return nil
	}
	//return NewGenericRepository("https://s3-ap-northeast-1.amazonaws.com/isaax-distro/vm/sd/0.1.0/isaax-box-sd.zip", "0.0.1", "virtualbox/sd/")
	return NewGenericRepository(rp.GetURL(), rp.GetVersion(), rp.Dir())
}

func CheckIoTItMD5(oss, arch, hash, release string) (string, error) {
	var h1, v string
	var checkMeth = "md5sums"
	var version = "version"

	var client http.Client
	resp, err := client.Get(IoTItRepo)
	if err != nil {
		return h1, err
	}
	defer resp.Body.Close()

	r := make(map[string]*json.RawMessage)
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return h1, err
	}
	if err := json.Unmarshal(*r[release], &r); err != nil {
		return h1, err
	}
	if err := json.Unmarshal(*r[checkMeth], &r); err != nil {
		return h1, err
	}
	if err := json.Unmarshal(*r[oss], &r); err != nil {
		return h1, err
	}
	if err := json.Unmarshal(*r[arch], &h1); err != nil {
		return h1, err
	}
	if err := json.Unmarshal(*r[version], &v); err != nil {
		return h1, err
	}

	if strings.EqualFold(hash, h1) {
		return "", nil
	}

	return v, nil
}
