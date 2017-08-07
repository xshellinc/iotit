package process

import (
	"time"
)

// Generic process interface
type Process interface {
	// Get process id
	Pid() int
	// Get parent process id
	PPid() int
	// Get binary name
	Binary() string
	// Get arguments
	Args() []string
	// Get executable path
	Executable() string
	// Get total virtual memory
	MemorySize() uint64
	// Get resident memory
	MemoryResident() uint64
	// Get shared memory
	MemoryShared() uint64
	// Refresh data
	Refresh() error
	// User time
	UserTime() time.Duration
	// System time
	SystemTime() time.Duration
}

// Get list of all available process
func Processes() (map[int]Process, error) {
	return processes()
}

// Find process by process id
func Find(pid int) (Process, error) {
	return find(pid)
}

// Find process by binary path
func FindByExecutable(path string) (map[int]Process, error) {
	return findByExecutable(path)
}
