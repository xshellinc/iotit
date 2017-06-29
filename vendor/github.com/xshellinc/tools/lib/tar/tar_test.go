package tar

import (
	"archive/tar"
	"os"
	"testing"
)

func Test_handleError(t *testing.T) {
	type args struct {
		_e error
	}
	tests := []struct {
		name string
		args args
	}{
	//
	}
	for _, tt := range tests {
		handleError(tt.args._e)
	}
}

func TestTarGzWrite(t *testing.T) {
	type args struct {
		_path string
		tw    *tar.Writer
		fi    os.FileInfo
	}
	tests := []struct {
		name string
		args args
	}{
	//
	}
	for _, tt := range tests {
		TarGzWrite(tt.args._path, tt.args.tw, tt.args.fi)
	}
}

func TestIterDirectory(t *testing.T) {
	type args struct {
		dirPath string
		tw      *tar.Writer
	}
	tests := []struct {
		name string
		args args
	}{
	//
	}
	for _, tt := range tests {
		IterDirectory(tt.args.dirPath, tt.args.tw)
	}
}

func TestTarGz(t *testing.T) {
	type args struct {
		outFilePath string
		inPath      string
	}
	tests := []struct {
		name string
		args args
	}{
	//
	}
	for _, tt := range tests {
		TarGz(tt.args.outFilePath, tt.args.inPath)
	}
}

func TestMakeTarBall(t *testing.T) {
	type args struct {
		targetFilePath string
		inputDirPath   string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test-tar-ball",
			args: args{
				targetFilePath: "/Users/tkila/app-test/test-app-js/test-app-js.tar.gz",
				inputDirPath:   "/Users/tkila/app-test/test-app-js/test-app-js/",
			},
		},
	}
	for _, tt := range tests {
		MakeTarBall(tt.args.targetFilePath, tt.args.inputDirPath)
	}
}

func TestTarit(t *testing.T) {
	type args struct {
		target string
		source string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test-tar-ball",
			args: args{
				target: "/tmp/",
				source: "/tmp/appjs",
			},
		},
	}
	for _, tt := range tests {
		if err := Tarit(tt.args.target, tt.args.source); (err != nil) != tt.wantErr {
			t.Errorf("%q. Tarit() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}
