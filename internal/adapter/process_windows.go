// process_windows.go — Windows-specific process group handling.
//
// Build tag: windows
//
// On Windows, CREATE_NEW_PROCESS_GROUP (0x00000200) creates a new process
// group. On timeout, taskkill /T /F /PID kills the process tree.

//go:build windows

package adapter

import (
	"os/exec"
	"strconv"
	"syscall"
)

// setProcessGroupAttr configures the command to run in a new process group.
//
// CREATE_NEW_PROCESS_GROUP (0x00000200) isolates the child process (cmd.exe)
// and its children (the CLI) into a separate process group. This allows
// killProcessGroup to target the entire tree with taskkill /T.
func setProcessGroupAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200}
}

// killProcessGroup kills the process tree for the given PID.
//
// Uses taskkill /T /F /PID:
//   /T   = kill child processes (tree)
//   /F   = force (no graceful shutdown)
//   /PID = target process ID
//
// This is best-effort — errors are silently ignored. If the process tree
// no longer exists (all processes already dead), taskkill returns an error
// which we ignore.
//
// Known limitation: if the parent process (cmd.exe) already exited before
// taskkill runs, taskkill /T may not find the orphaned children because it
// searches for children of the specified PID. This is a Windows limitation;
// on Unix, the process group approach is more robust.
func killProcessGroup(pid int) {
	_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(pid)).Run()
}
