// +build darwin

//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package plot

import (
	tm "github.com/nsf/termbox-go"
)

type plotter struct {
}

func (p plotter) HideCursor() {
	tm.SetCursor(0, 0)
}

func (p plotter) BlockWindowResize() {
}
