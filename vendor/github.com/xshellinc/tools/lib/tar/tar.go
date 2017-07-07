package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/xshellinc/tools/lib/help"
)

var s string = help.Separator()

func handleError(_e error) {
	if _e != nil {
		log.Debug(_e)
	}
}

func TarGzWrite(_path string, tw *tar.Writer, fi os.FileInfo) {
	fr, err := os.Open(_path)
	handleError(err)
	defer fr.Close()

	h := new(tar.Header)
	h.Name = fr.Name()
	h.Size = fi.Size()
	h.Mode = int64(fi.Mode())
	h.ModTime = fi.ModTime()

	err = tw.WriteHeader(h)
	handleError(err)

	_, err = io.Copy(tw, fr)
	handleError(err)
}

func IterDirectory(dirPath string, tw *tar.Writer) {
	dir, err := os.Open(dirPath)
	handleError(err)
	defer dir.Close()
	fis, err := dir.Readdir(0)
	handleError(err)
	for _, fi := range fis {
		curPath := dirPath + s + fi.Name()
		if fi.IsDir() {
			//TarGzWrite( curPath, tw, fi )
			IterDirectory(curPath, tw)
		} else {
			log.Debug(curPath)
			TarGzWrite(curPath, tw, fi)
		}
	}
}

func TarGz(outFilePath string, inPath string) {
	// file write
	fw, err := os.Create(outFilePath)
	handleError(err)
	defer fw.Close()

	// gzip write
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// tar write
	tw := tar.NewWriter(gw)
	defer tw.Close()

	IterDirectory(inPath, tw)
}

//TODO: This function takes absolute path for each file
func MakeTarBall(targetFilePath string, inputDirPath string) {
	TarGz(targetFilePath, strings.TrimRight(inputDirPath, s))
}

func Tarit(target, source string) error {
	filename := filepath.Base(source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar", filename))
	tarfile, err := os.Create(target)

	if err != nil {
		return err
	}
	defer tarfile.Close()
	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()
	info, err := os.Stat(source)
	if err != nil {
		return nil
	}
	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}
	return filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}
			if err := tarball.WriteHeader(header); err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
}
