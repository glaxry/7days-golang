//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package geebolt

import (
	"os"

	"golang.org/x/sys/unix"
)

type mmapState struct{}

func mapFile(file *os.File, size int) ([]byte, mmapState, error) {
	data, err := unix.Mmap(int(file.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	return data, mmapState{}, err
}

func syncMappedFile(data []byte, _ mmapState) error {
	return unix.Msync(data, unix.MS_SYNC)
}

func unmapFile(data []byte, _ mmapState) error {
	return unix.Munmap(data)
}
