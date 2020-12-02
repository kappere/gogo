package util

import (
	"runtime"
	"strings"
)

func GetCurrentThreadId() int {
	sysType := runtime.GOOS

	if strings.ToLower(sysType) == "windows" {
		// var user32 *syscall.DLL
		// var GetCurrentThreadId *syscall.Proc
		// var err error

		// user32, err = syscall.LoadDLL("Kernel32.dll")
		// if err != nil {
		// 	fmt.Printf("syscall.LoadDLL fail: %v\n", err.Error())
		// 	return 0
		// }
		// GetCurrentThreadId, err = user32.FindProc("GetCurrentThreadId")
		// if err != nil {
		// 	fmt.Printf("user32.FindProc fail: %v\n", err.Error())
		// 	return 0
		// }

		// var pid uintptr
		// pid, _, err = GetCurrentThreadId.Call()

		// return int(pid)
	}
	return -1
}
