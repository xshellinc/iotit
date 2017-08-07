package dialogs

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var Handler DialogHandler

type DialogHandler struct {
	Reader io.Reader
}

func sumLength(arr []int64) int64 {
	var c int64
	for _, v := range arr {
		c += v
	}

	return c
}

func initDialogHandler(a *assert.Assertions, s string) *os.File {
	logrus.SetOutput(ioutil.Discard)

	in, err := ioutil.TempFile("", "")
	a.NoError(err, "Error running test, ioutil.TempFile")

	_, err = io.WriteString(in, s)
	a.NoError(err, "Error running test, io.WriteString(TempFile, ...)")

	Handler = DialogHandler{in}

	return in
}

func TestYesNoDialog(t *testing.T) {
	inps := [5]string{
		"y\n",
		"n\n",
		"yes\n",
		"no\n",
		"asd\ny\n",
	}
	lens := [6]int64{
		2, // starting from 0
		2,
		4,
		6,
		3,
	}

	a := assert.New(t)
	in := initDialogHandler(a, strings.Join(inps[:], ""))
	defer in.Close()

	for i := 0; i < 2; i++ {
		t.Log(sumLength(lens[:i+1]))
		_, err := in.Seek(sumLength(lens[:i]), os.SEEK_SET)
		a.NoError(err, "Error running test, TempFile.Seek")

		if i%2 == 0 {
			a.Equal(true, YesNoDialog(""))
			continue
		}

		a.Equal(false, YesNoDialog(""))
	}
}

func TestGetSingleAnswer(t *testing.T) {
	param := "anything"
	inp := param + "\n"

	a := assert.New(t)
	in := initDialogHandler(a, inp)
	defer in.Close()

	_, err := in.Seek(0, os.SEEK_SET)
	a.NoError(err, "Error running test, TempFile.Seek")

	a.Equal(param, GetSingleAnswer(""))
}

func TestGetSingleAnswer2(t *testing.T) {
	param := "192.168.1.1"
	inp := "\nasd\n" + param + "\n"

	a := assert.New(t)
	in := initDialogHandler(a, inp)
	defer in.Close()

	_, err := in.Seek(0, os.SEEK_SET)
	a.NoError(err, "Error running test, TempFile.Seek")

	a.Equal(param, GetSingleAnswer("", EmptyStringValidator, IpAddressValidator))
}

func TestSelectOneDialog(t *testing.T) {
	// Testing a 3 => index out of b, 1 => 0 success
	// Testing b 3 => index out of b, -1 => index out of b, 2 => (1) success
	inp := "3\n1\naws\n-1\n2\n"
	var seek int64 = 4

	a := assert.New(t)
	in := initDialogHandler(a, inp)
	defer in.Close()

	_, err := in.Seek(0, os.SEEK_SET)
	a.NoError(err, "Error running test, TempFile.Seek")

	a.Equal(0, SelectOneDialog("", []string{"", ""}))

	_, err = in.Seek(seek, os.SEEK_SET)
	a.NoError(err, "Error running test, TempFile.Seek")

	a.Equal(1, SelectOneDialog("", []string{"", ""}))
}

func TestSelectOneDialogCrash(t *testing.T) {
	if os.Getenv("CRASHABLE") == "1" {
		inp := "3\nnaws\n-1\n"

		a := assert.New(t)
		in := initDialogHandler(a, inp)
		defer in.Close()

		_, err := in.Seek(0, os.SEEK_SET)
		a.NoError(err, "Error running test, TempFile.Seek")

		SelectOneDialog("", []string{""})

		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSelectOneDialogCrash")
	cmd.Env = append(os.Environ(), "CRASHABLE=1")
	err := cmd.Run()

	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status != 0", err)
}

func TestCreateValidatorFn(t *testing.T) {
	fn := func(a string) error {
		if a == "" {
			return nil
		}
		return errors.New("err")
	}

	a := assert.New(t)

	runFun := CreateValidatorFn(fn)

	a.Equal(true, runFun(""))
	a.Equal(false, runFun("anything"))
}

func TestEmptyStringValidator(t *testing.T) {
	a := assert.New(t)
	a.Equal(false, EmptyStringValidator(""))
	a.Equal(true, EmptyStringValidator("anything"))
}

func TestIpAddressValidator(t *testing.T) {
	a := assert.New(t)
	a.Equal(false, YesNoValidator(""))
	a.Equal(false, IpAddressValidator("text"))
	a.Equal(false, IpAddressValidator("123"))
	a.Equal(false, IpAddressValidator("192.168.1.257"))
	a.Equal(false, IpAddressValidator("192.168.1.1230"))
	a.Equal(true, IpAddressValidator("192.168.1.123"))
}

func TestYesNoValidator(t *testing.T) {
	a := assert.New(t)
	a.Equal(false, YesNoValidator(""))
	a.Equal(false, YesNoValidator("ye"))
	a.Equal(true, YesNoValidator("yes"))
	a.Equal(true, YesNoValidator("y"))
	a.Equal(true, YesNoValidator("n"))
	a.Equal(true, YesNoValidator("no"))
}

func TestSpecialCharacterValidator(t *testing.T) {
	a := assert.New(t)
	fn := SpecialCharacterValidator("/*/^", false)

	a.Equal(true, fn("a*fb"))
	a.Equal(true, fn("a*^b"))
	a.Equal(true, fn("*^"))
	a.Equal(true, fn("a^asdf^^^"))
	a.Equal(true, fn("a^asdf^^^"))
	a.Equal(false, fn("textaasdf"))

	fn = SpecialCharacterValidator("/*/^", true)

	a.Equal(false, fn("a*fb"))
	a.Equal(false, fn("a*^b"))
	a.Equal(false, fn("*^"))
	a.Equal(false, fn("a^asdf^^^"))
	a.Equal(false, fn("a^asdf^^^"))
	a.Equal(true, fn("textaasdf"))
}
