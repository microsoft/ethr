//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

type OSStats interface {
	GetNetDevStats() ([]EthrNetDevStat, error)
	GetTCPStats() (EthrTCPStat, error)
}

// GetOSStats returns Ethr-relevant OS statistics, dependant on the build flag
func GetOSStats() OSStats {
	return osStats{}
}
