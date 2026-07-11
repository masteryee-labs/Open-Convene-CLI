// process_unix.go — Unix-specific process group handling.
//
// Build tag: !windows (Linux, macOS, BSD, etc.)
//
// On Unix, Setpgid creates a new process group so that timeout can kill
// the entire group (shell + CLI grandchild) with kill(-pgid, SIGKILL).

//go:build !windows

package adapter

import (
	"os/exec"
	"syscall"
)

// setProcessGroupAttr configures the command to run in a new process group.
//
// Setpgid: true puts the shell (sh) and its children (the CLI) in a new
// process group. This allows killProcessGroup to send SIGKILL to every
// process in the group via kill(-pgid, SIGKILL).
func setProcessGroupAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup sends SIGKILL to the entire process group.
//
// The negative PID (-pid) tells the kernel to signal every process in the
// process group whose ID equals pid. Even after the process group leader
// (sh) is killed by Go's context cancel, the group persists while
// grandchild processes (the CLI) are still alive, so this call kills them.
//
// This is best-effort — errors are silently ignored. If the group no
// longer exists (all processes already dead), kill returns ESRCH which
// we ignore.
func killProcessGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
