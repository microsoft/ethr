//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package plot

// Plotter is a generic plotter interface for all supported OS families
type Plotter interface {
	BlockWindowResize()
	HideCursor()
}

// GetPlotter returns a new, os-specific plotter depending on the build flags
func GetPlotter() Plotter {
	return plotter{}
}
