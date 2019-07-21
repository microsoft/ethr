//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package cmd

import "fmt"

// EthrUsage prints the command-line usage text
func EthrUsage(gVersion string) {
	fmt.Println("\nEthr - A comprehensive network performance measurement tool.")
	fmt.Println("Version: " + gVersion)
	fmt.Println("It supports 4 modes. Usage of each mode is described below:")

	fmt.Println("\nCommon Parameters")
	fmt.Println("================================================================================")
	PrintFlagUsage("h", "", "Help")
	PrintFlagUsage("no", "", "Disable logging to file. Logging to file is enabled by default.")
	PrintFlagUsage("o", "<filename>", "Name of log file. By default, following file names are used:",
		"Server mode: 'ethrs.log'",
		"Client mode: 'ethrc.log'",
		"External server mode: 'ethrxs.log'",
		"External client mode: 'ethrxc.log'")
	PrintFlagUsage("debug", "", "Enable debug information in logging output.")
	PrintFlagUsage("4", "", "Use only IP v4 version")
	PrintFlagUsage("6", "", "Use only IP v6 version")

	fmt.Println("\nMode: Server")
	fmt.Println("================================================================================")
	PrintServerUsage()
	PrintFlagUsage("ui", "", "Show output in text UI.")
	PrintPortUsage()

	fmt.Println("\nMode: Client")
	fmt.Println("================================================================================")
	PrintClientUsage()
	PrintFlagUsage("r", "", "For Bandwidth tests, send data from server to client.")
	PrintDurationUsage()
	PrintThreadUsage()
	PrintNoConnStatUsage()
	PrintBufLenUsage()
	PrintProtocolUsage()
	PrintIgnoreCertUsage()
	PrintPortUsage()
	PrintTestType()
	PrintIterationUsage()

	fmt.Println("\nMode: External Server")
	fmt.Println("================================================================================")
	PrintModeUsage()
	PrintServerUsage()
	PrintExtPortUsage()

	fmt.Println("\nMode: External Client")
	fmt.Println("================================================================================")
	PrintModeUsage()
	PrintExtClientUsage()
	PrintDurationUsage()
	PrintThreadUsage()
	PrintNoConnStatUsage()
	PrintBufLenUsage()
	PrintExtProtocolUsage()
	PrintExtTestType()
	PrintGapUsage()
}
