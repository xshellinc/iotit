package inotify

import (
	"golang.org/x/sys/unix"
	"os"
	"unsafe"
)

const (
	// InAccess event generated when file was accessed (e.g., read(2), execve(2))
	InAccess = unix.IN_ACCESS
	// InAttrib event generated when metadata changed -- for example, permissions (e.g., chmod(2)), timestamps (e.g., utimensat(2)),
	// extended attributes (setxattr(2)), link count (since Linux
	// 2.6.25; e.g., for the target of link(2) and for unlink(2)), and user/group ID (e.g., chown(2))
	InAttrib = unix.IN_ATTRIB
	// InCloseWrite event generated when file opened for writing was closed
	InCloseWrite = unix.IN_CLOSE_WRITE
	// InCloseNowrite event generated when file or directory not opened for writing was closed
	InCloseNowrite = unix.IN_CLOSE_NOWRITE
	// InCreate event generated when file/directory created in watched directory (e.g., open(2) O_CREAT, mkdir(2), link(2), symlink(2), bind(2)
	// on a UNIX domain socket)
	InCreate = unix.IN_CREATE
	// InDelete event generated when file/directory deleted from watched directory
	InDelete = unix.IN_DELETE
	// InDeleteSelf event generated when watched file/directory was itself deleted. (This event also occurs if an object is moved to another filesystem,
	// since mv(1) in effect copies the file to the other filesystem and then deletes it from the original filesystem.)
	// In addition, an IN_IGNORED event will subsequently be generated for the watch descriptor
	InDeleteSelf = unix.IN_DELETE_SELF
	// InModify event generated when file was modified (e.g., write(2), truncate(2))
	InModify = unix.IN_MODIFY
	// InMoveSelf event generated when watched file/directory was itself moved
	InMoveSelf = unix.IN_MOVE_SELF
	// InMovedFrom event generated for the directory containing the old filename when a file is renamed
	InMovedFrom = unix.IN_MOVED_FROM
	// InMovedTo event generated for the directory containing the new filename when a file is renamed
	InMovedTo = unix.IN_MOVED_TO
	// InOpen event generated when file or directory was opened
	InOpen = unix.IN_OPEN
)

// Inotify main object
type Inotify struct {
	C     <-chan Event
	fp    *os.File
	rPipe *os.File
	wPipe *os.File
	sem   chan struct{}
}

// Event represents inotify event
type Event struct {
	Watch  Watch
	Mask   uint32
	Name   string
	Cookie uint32
}

// Watch represents single watch associated with inotify object
type Watch struct {
	fd int
	wd int32
}

// New returns new empty Inotify object
func New() (*Inotify, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, err
	}

	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	c := make(chan Event, 1024)

	in := &Inotify{
		fp:    os.NewFile(uintptr(fd), "inotify_object"),
		sem:   make(chan struct{}),
		rPipe: rPipe,
		wPipe: wPipe,
		C:     c,
	}

	go in.loop(c)

	return in, nil
}

func (i *Inotify) loop(c chan Event) {
	var tmp unix.InotifyEvent
	sz := unsafe.Sizeof(tmp)

	buf := make([]byte, sz+unix.NAME_MAX)

	pollFds := []unix.PollFd{
		{
			Fd:     int32(i.rPipe.Fd()),
			Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR,
		},
		{
			Fd:     int32(i.fp.Fd()),
			Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR,
		},
	}

	for {
		n, _ := unix.Poll(pollFds, -1)

		if n == 0 {
			continue
		} else if n < 0 {
			break
		}

		if pollFds[0].Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) != 0 ||
			pollFds[1].Revents&(unix.POLLHUP|unix.POLLERR) != 0 {
			// Pipe closed or error occurred
			break

		} else if pollFds[1].Revents&unix.POLLIN != 0 {
			// Inotify

			n, err := i.fp.Read(buf)
			if err != nil {
				break
			}

			var offs uintptr
			for offs < uintptr(n) {
				iev := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offs]))

				ev := Event{
					Watch: Watch{
						wd: iev.Wd,
						fd: int(i.fp.Fd()),
					},
					Mask:   iev.Mask,
					Cookie: iev.Cookie,
				}

				if iev.Len != 0 {
					ev.Name = string(buf[offs+sz : offs+sz+uintptr(iev.Len)-1])
				}

				c <- ev

				offs += sz + uintptr(iev.Len)
			}
		}
	}

	close(c)

	i.rPipe.Close()
	i.fp.Close()

	i.sem <- struct{}{}
}

// Add adds watched path
func (i *Inotify) Add(path string, mask uint32) (Watch, error) {
	wd, err := unix.InotifyAddWatch(int(i.fp.Fd()), path, mask)
	if err != nil {
		return Watch{}, err
	}

	return Watch{
		fd: int(i.fp.Fd()),
		wd: int32(wd),
	}, nil
}

// Close destroys Inotify object
func (i *Inotify) Close() error {
	if err := i.wPipe.Close(); err != nil {
		return err
	}

	// Drain channel to unblock writer (unlikely, but to be sure)
	for len(i.C) != 0 {
		_, _ = <-i.C
	}

	<-i.sem

	return nil
}

// Remove removes watch from it's parent Inotify object
func (w Watch) Remove() error {
	_, err := unix.InotifyRmWatch(w.fd, uint32(w.wd))
	return err
}
