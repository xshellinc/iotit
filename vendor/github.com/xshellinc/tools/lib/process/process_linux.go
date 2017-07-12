package process

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// linux-only musl libc always uses this value, see musl/src/conf/sysconf.c
	_SC_CLK_TCK = 100
)

type linuxProcessMem struct {
	// Size represents total program size aka VmSize
	Size uint64 `json:"size"`
	// RSS represents resident set size aka VmRSS
	RSS uint64 `json:"rss"`
	// Shared represents number of resident shared pages
	Shared uint64 `json:"shared"`
}

type linuxProcessTime struct {
	// User represents user time in nanoseconds
	User uint64 `json:"user"`
	// System represents system time in nanoseconds
	System uint64 `json:"system"`
}

type linuxProcess struct {
	// PID represents process ID
	PID int `json:"pid"`

	// PPID represents parent process ID
	PPID int `json:"ppid"`

	// S represents process state
	S rune `json:"s"`

	// PGRP represents process group ID
	PGRP int `json:"pgrp"`

	// SID represents session ID
	SID int `json:"pgrp"`

	// CmdArgs represents arguments passed
	CmdArgs []string `json:"cmdArgs"`

	// Bin represents binary file name
	Bin string `json:"bin"`

	// Exe represents full path to executable
	Exe string `json:"executable"`

	// Memory represents process memory usage info
	Memory linuxProcessMem `json:"memory"`

	// Time represents process CPU time
	Time linuxProcessTime `json:"time"`
}

func processes() (map[int]Process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	results := make(map[int]Process)

	fis, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		name := fi.Name()
		if name[0] < '0' || name[0] > '9' {
			continue
		}

		pid, err := strconv.ParseInt(name, 10, 0)
		if err != nil {
			continue
		}

		p, err := newLinuxProcess(int(pid))
		if err != nil {
			continue
		}

		results[int(pid)] = p
	}

	return results, nil
}

func find(pid int) (Process, error) {
	dir := fmt.Sprintf("/proc/%d", pid)
	_, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	return newLinuxProcess(pid)
}

func findByExecutable(path string) (map[int]Process, error) {
	d, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer d.Close()

	results := make(map[int]Process)

	fis, err := d.Readdir(-1)
	if err != nil {
		return nil, err
	}

	for _, fi := range fis {
		if !fi.IsDir() {
			continue
		}

		name := fi.Name()
		if name[0] < '0' || name[0] > '9' {
			continue
		}

		pid, err := strconv.ParseInt(name, 10, 0)
		if err != nil {
			continue
		}

		p := linuxProcess{PID: int(pid)}
		if err := p.executable(); err != nil {
			continue
		}

		if p.Exe == path {
			if err := p.Refresh(); err != nil {
				continue
			}

			results[int(pid)] = &p
		}
	}

	return results, nil
}

func newLinuxProcess(pid int) (*linuxProcess, error) {
	p := &linuxProcess{PID: pid}

	if err := p.Refresh(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *linuxProcess) Pid() int {
	return p.PID
}

func (p *linuxProcess) PPid() int {
	return p.PPID
}

func (p *linuxProcess) Binary() string {
	return p.Bin
}

func (p *linuxProcess) Args() []string {
	return p.CmdArgs
}

func (p *linuxProcess) Executable() string {
	return p.Exe
}

func (p *linuxProcess) MemorySize() uint64 {
	return p.Memory.Size
}

func (p *linuxProcess) MemoryResident() uint64 {
	return p.Memory.RSS
}

func (p *linuxProcess) MemoryShared() uint64 {
	return p.Memory.Shared
}

func (p *linuxProcess) UserTime() time.Duration {
	return time.Duration(p.Time.User) * time.Nanosecond
}

func (p *linuxProcess) SystemTime() time.Duration {
	return time.Duration(p.Time.System) * time.Nanosecond
}

func (p *linuxProcess) args() error {
	cmdPath := fmt.Sprintf("/proc/%d/cmdline", p.PID)
	dataBytes, err := ioutil.ReadFile(cmdPath)
	if err != nil {
		return err
	}
	p.CmdArgs = strings.FieldsFunc(string(dataBytes), func(r rune) bool { return r == 0 })
	return nil
}

func (p *linuxProcess) executable() error {
	path := fmt.Sprintf("/proc/%d/exe", p.PID)

	var err error
	p.Exe, err = os.Readlink(path)
	return err
}

func (p *linuxProcess) memory() error {
	path := fmt.Sprintf("/proc/%d/statm", p.PID)
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var size, res, shared uint64
	if n, _ := fmt.Sscanf(string(data), "%d %d %d", &size, &res, &shared); n != 3 {
		return errors.New("Error parsing " + path)
	}

	pgs := uint64(os.Getpagesize())

	p.Memory = linuxProcessMem{
		Size:   size * pgs,
		RSS:    res * pgs,
		Shared: shared * pgs,
	}

	return nil
}

func (p *linuxProcess) Refresh() error {
	statPath := fmt.Sprintf("/proc/%d/stat", p.PID)

	dataBytes, err := ioutil.ReadFile(statPath)
	if err != nil {
		return err
	}

	fields := fieldsQuoted(string(dataBytes), "(", ")")
	if len(fields) < 15 {
		return errors.New("Error parsing " + statPath)
	}

	p.Bin = fields[1]
	p.S = rune(fields[2][0])

	p.PPID, _ = strconv.Atoi(fields[3])
	p.PGRP, _ = strconv.Atoi(fields[4])
	p.SID, _ = strconv.Atoi(fields[5])

	utime, _ := strconv.ParseUint(fields[13], 10, 64)
	stime, _ := strconv.ParseUint(fields[14], 10, 64)

	p.Time.User = utime * 1e9 / _SC_CLK_TCK
	p.Time.System = stime * 1e9 / _SC_CLK_TCK

	// obtain binary (ignore errors)
	p.executable()

	// obtain memory info
	if err := p.memory(); err != nil {
		return err
	}

	// obtain args
	return p.args()
}

func fieldsQuoted(str, opensep, closesep string) []string {
	var out []string

	quote := false
	fieldStart := -1

	for i, r := range str {
		if quote {
			if strings.ContainsRune(closesep, r) {
				out = append(out, str[fieldStart:i])
				fieldStart = -1
				quote = false
			}

			continue
		}

		switch {
		case strings.ContainsRune(opensep, r):
			if fieldStart >= 0 {
				out = append(out, str[fieldStart:i])
			}

			fieldStart = i + 1
			quote = true

		case unicode.IsSpace(r):
			if fieldStart >= 0 {
				out = append(out, str[fieldStart:i])
				fieldStart = -1
			}

		default:
			if fieldStart < 0 {
				fieldStart = i
			}
		}
	}

	if fieldStart >= 0 {
		out = append(out, str[fieldStart:len(str)])
	}

	return out
}
