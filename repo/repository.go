package repo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/tools/lib/help"
	"gopkg.in/cheggaaa/pb.v1"
)

// S3Bucket stores default S3 bucket path
const S3Bucket = "https://cdn.isaax.io/isaax-distro/versions.json"

// IoTItRepo stores default iotit repo path
const IoTItRepo = "https://cdn.isaax.io/iotit/version.json"

// Releases
const (
	Latest = "latest"
	Stable = "stable"
)

// baseDir is a directory of iotit related files and configurations
var baseDir = filepath.Join(help.UserHomeDir(), ".iotit")

// VboxDir is a directory of virtualboxes
var VboxDir = filepath.Join(baseDir, "virtualbox")

// ImageDir is a directory of the flashing images
var ImageDir = filepath.Join(baseDir, "images")

func init() {
	help.CreateDir(baseDir)
	help.CreateDir(ImageDir)
	help.CreateDir(VboxDir)
	help.CreateDir(filepath.Join(help.UserHomeDir(), "VirtualBox VMs"))
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

// GenericRepository is so generic
type GenericRepository struct {
	Version   string
	URL       string
	Directory string
}

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

// VMRepo is a configuration entry for VM
type VMRepo struct {
	VMs struct {
		VM struct {
			Version string `json:"version"`
			URL     string `json:"url"`
			MD5Sum  string `json:"md5sum"`
		} `json:"vm-iotit"`
	} `json:"vms"`
}

// GetVersion of VM
func (v VMRepo) GetVersion() string {
	return v.VMs.VM.Version
}

// GetURL of VM
func (v VMRepo) GetURL() string {
	return v.VMs.VM.URL
}

// Dir of VM
func (VMRepo) Dir() string {
	return VboxDir
}

// Name of VM
func (v VMRepo) Name() string {
	tokens := strings.Split(v.VMs.VM.URL, "/")
	return tokens[len(tokens)-1]
}

// NewRepositoryVM creates new repository for specified VM type
func NewRepositoryVM() (Repository, error) {
	var (
		client http.Client
		repo   VMRepo
	)
	resp, err := client.Get(S3Bucket)
	if err != nil {
		log.Error("Could not make GET request to url:", S3Bucket, " error msg:", err.Error())
		fmt.Println("[-] Could not connect to S3 bucket")
		return nil, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&repo); err != nil {
		log.Error("Could not unmarshall json struct ", "error msg:", err.Error())
		return nil, err
	}

	return repo, nil
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

// DownloadNewVersion downloads the latest version based on the current release and skips this step if up to date
func DownloadNewVersion(name, version, dst string) (string, error) {
	zipMethod := "zip"
	if runtime.GOOS == "linux" {
		zipMethod = "tar.gz"
	}

	fileName := fmt.Sprintf("%s_%s_%s_%s", name, version, runtime.GOOS, runtime.GOARCH)

	_, version, err := GetIoTItVersionMD5(runtime.GOOS, runtime.GOARCH, version)
	if err != nil || version == "" {
		return "", err
	}

	url := fmt.Sprintf("https://cdn.isaax.io/%s/%s/%s/%s.%s", name, currentRelease(version), runtime.GOOS, fileName, zipMethod)

	wg := &sync.WaitGroup{}
	imgName, bar, err := help.DownloadFromUrlWithAttemptsAsync(url, dst, 5, wg)
	if err != nil {
		return fileName, err
	}
	bar.Prefix(fmt.Sprintf("[+] Download %-15s", imgName))
	bar.Start()
	wg.Wait()
	bar.Finish()
	time.Sleep(time.Second)

	fmt.Println("[+] Extracting into ", dst)
	if runtime.GOOS == "linux" {
		if err := exec.Command("tar", "xvf", dst+help.Separator()+fileName, "-C", dst).Run(); err != nil {
			fmt.Println("[-] ", err)
			return fileName, err
		}
	} else if err := exec.Command("unzip", "-o", dst+help.Separator()+fileName, "-d", dst).Run(); err != nil {
		fmt.Println("[-] ", err)
		return fileName, err
	}

	return fileName, nil
}

// currentRelease detects whether release is stable or latest
func currentRelease(version string) (release string) {
	r := Latest
	match, _ := regexp.Compile(`^[\d|_]+\.[\d|_]+\.[\d|_]+$`)
	if match.MatchString(version) {
		r = Stable
	}

	return r
}

// getVersionLexem parses string lexems into comparable parts
func getVersionLexem(token string, seps ...string) []string {
	var lexs []string
	for i, sep := range seps {
		if i == 0 {
			lexs = strings.Split(token, sep)
			continue
		}

		var tmp []string
		for _, lex := range lexs {
			tmp = append(tmp, strings.Split(lex, sep)...)
		}

		lexs = tmp
	}

	return lexs
}

// IsVersionUpToDate checks if version is up to date
func IsVersionUpToDate(v1, v2 string) (bool, error) {
	vlex1 := getVersionLexem(v1, ".", "_", "-")
	vlex2 := getVersionLexem(v2, ".", "_", "-")

	for i := 0; i < len(vlex1) && i < len(vlex2); i++ {
		n1, err := strconv.Atoi(vlex1[i])
		if err != nil {
			return false, err
		}

		n2, err := strconv.Atoi(vlex2[i])
		if err != nil {
			return false, err
		}

		if n1 == n2 {
			continue
		}

		return n1 > n2, nil
	}

	// not reachable
	return false, nil
}

// GetIoTItVersionMD5 gets the latest version from the repo and checks if the new version is available
func GetIoTItVersionMD5(oss, arch, version string) (hash string, repoVersion string, err error) {
	var checkMethKey = "md5sums"
	var versionKey = "version"

	var client http.Client
	resp, err := client.Get(IoTItRepo)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	r := make(map[string]*json.RawMessage)
	if err = json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return
	}
	if err = json.Unmarshal(*r[currentRelease(version)], &r); err != nil {
		return
	}
	if err = json.Unmarshal(*r[checkMethKey], &r); err != nil {
		return
	}
	if err = json.Unmarshal(*r[oss], &r); err != nil {
		return
	}
	if err = json.Unmarshal(*r[arch], &hash); err != nil {
		return
	}
	if err = json.Unmarshal(*r[versionKey], &repoVersion); err != nil {
		return
	}
	var bl bool
	if bl, err = IsVersionUpToDate(version, repoVersion); bl {
		repoVersion = ""
		return
	}

	return
}
