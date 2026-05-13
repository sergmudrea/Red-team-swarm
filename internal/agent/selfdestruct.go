package agent

import (
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

// SelfDestruct removes the running executable and terminates the process.
// On Windows it schedules the file for deletion at the next reboot, then exits.
// On other platforms it deletes the file immediately and calls os.Exit(0).
func SelfDestruct() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		moveFileEx := kernel32.NewProc("MoveFileExW")
		exePathUTF16, err := syscall.UTF16PtrFromString(exePath)
		if err != nil {
			return err
		}
		const MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
		ret, _, _ := moveFileEx.Call(
			uintptr(unsafe.Pointer(exePathUTF16)),
			0,
			MOVEFILE_DELAY_UNTIL_REBOOT,
		)
		if ret == 0 {
			return syscall.GetLastError()
		}
		os.Exit(0)
	default:
		if err := os.Remove(exePath); err != nil {
			return err
		}
		os.Exit(0)
	}
	return nil
}
