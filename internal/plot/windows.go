// +build windows

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package plot

import (
	"syscall"

	tm "github.com/nsf/termbox-go"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	procGetSystemMenu    = user32.NewProc("GetSystemMenu")
	procDeleteMenu       = user32.NewProc("DeleteMenu")
)

type plotter struct {
}

func (p plotter) HideCursor() {
	tm.HideCursor()
}

const (
	MFByCommand = 0x00000000
	SCMaximize  = 0xF030
	SCSize      = 0xF000
)

func (p plotter) BlockWindowResize() {
	h, _, err := syscall.Syscall(procGetConsoleWindow.Addr(), 0, 0, 0, 0)
	if err != 0 {
		return
	}

	sysMenu, _, err := syscall.Syscall(procGetSystemMenu.Addr(), 2, h, 0, 0)
	if err != 0 {
		return
	}

	syscall.Syscall(procDeleteMenu.Addr(), 3, sysMenu, SCMaximize, MFByCommand)
	syscall.Syscall(procDeleteMenu.Addr(), 3, sysMenu, SCSize, MFByCommand)
}
