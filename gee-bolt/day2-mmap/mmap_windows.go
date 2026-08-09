//go:build windows

package geebolt

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type mmapState struct {
	handle windows.Handle
	addr   uintptr
}

func mapFile(file *os.File, size int) ([]byte, mmapState, error) {
	handle, err := windows.CreateFileMapping(
		windows.Handle(file.Fd()),
		nil,
		windows.PAGE_READWRITE,
		0,
		uint32(size),
		nil,
	)
	if err != nil {
		return nil, mmapState{}, err
	}
	addr, err := windows.MapViewOfFile(handle, windows.FILE_MAP_WRITE, 0, 0, uintptr(size))
	if err != nil {
		_ = windows.CloseHandle(handle)
		return nil, mmapState{}, err
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(addr)), size)
	return data, mmapState{handle: handle, addr: addr}, nil
}

func syncMappedFile(data []byte, mapping mmapState) error {
	return windows.FlushViewOfFile(mapping.addr, uintptr(len(data)))
}

func unmapFile(_ []byte, mapping mmapState) error {
	return errors.Join(
		windows.UnmapViewOfFile(mapping.addr),
		windows.CloseHandle(mapping.handle),
	)
}
