 # Ethr [![Build Status](https://travis-ci.org/Microsoft/ethr.svg?branch=master)](https://travis-ci.org/Microsoft/ethr)

Ethr is a cross platform network performance measurement tool written in golang. The goal of this project is to provide a native tool for comprehensive network performance measurements of bandwidth, connections/s, packets/s, latency, loss & jitter, across multiple protocols such as TCP, UDP, HTTP, HTTPS, and across multiple platforms such as Windows, Linux and other Unix systems.

<p align="center">
  <img alt="Ethr server in action" src="https://user-images.githubusercontent.com/44273634/49815752-506f0000-fd21-11e8-954e-d587e79c5d85.png">
</p>

Ethr takes inspiration from existing open source network performance tools and builds upon those ideas. For Bandwidth measurement, it is similar to iPerf3, for TCP & UDP traffic. iPerf3 has many more options for doing such as throttled testing, richer feature set, while Ethr has support for multiple threads, that allows it to scale to 1024 or even higher number of connections, multiple clients communication to a single server etc. For latency measurements, it is similar to latte on Windows or sockperf on Linux.

Ethr provides more test measurements as compared to other tools, e.g. it provides measurements for bandwidth, connections/s, packets/s, latency, and TCP connection setup latency, all in a single tool. In the future, there are plans to add more features (hoping for others to contribute) as well as more protocol support to make it a comprehensive tool for network performance measurements.

Ethr is natively cross platform, thanks to golang, as compared to compiling via an abstraction layer like cygwin that may limit functionality. It hopes to unify performance measurement by combining the functionality of tools like iPerf3, ntttcp, psping, sockperf, and latte and offering a single tool across multiple platforms and multiple protocols.

# Download

https://github.com/Microsoft/ethr/releases/latest

# Installation

Note: go version 1.11 or higher is required building it from the source.

## Building from Source

We use go-module to manage Ethr dependencies. for more information please check [how to use go-modules!](https://github.com/golang/go/wiki/Modules#how-to-use-modules)

```
git clone https://github.com/Microsoft/ethr.git
cd ethr
go get ./...
go build
```

If Ethr is cloned inside of the `$GOPATH/src` tree, please make sure you invoke the `go` command with `GO111MODULE=on`!

## Docker

Build image using command: 
```
docker build -t microsoft/ethr .
```

Make binary:

**Linux**
```
docker run -e GOOS=linux -v $(pwd):/out microsoft/ethr make build-docker
```

**Windows**

```
docker run -e BINARY_NAME=ethr.exe -e GOOS=windows -v $(pwd):/out microsoft/ethr make build-docker
```

**OS X**
```
docker run -e BINARY_NAME=ethr -e GOOS=darwin -v $(pwd):/out microsoft/ethr make build-docker
```

## Using go get

```
go get github.com/Microsoft/ethr
```

## Using ArchLinux AUR

Assuming you are using [`yay`](https://aur.archlinux.org/packages/yay/) (https://github.com/Jguer/yay):

```
yay -S ethr
```
# Publishing Nuget package
Follow the topic Building from Source to build ethr.exe

Modify ethr.nuspec to add new release version
```
vim ethr.nuspec
```
Create a nuget package(like Ethr.0.2.1.nupkg)
```
nuget.exe pack ethr.nuspec
```
Upload the package to nuget.org.

# Usage

## Simple Usage
Help:
```
ethr -h
```

Server:
```
ethr -s
```

Server with Text UI:
```
ethr -s -ui
```

Client:
```
ethr -c <server ip>
```

Examples:
```
// Start server
ethr -s

// Start client for default (bandwidth) test measurement using 1 thread
ethr -c localhost

// Start bandwidth test using 8 threads
ethr -c localhost -n 8

// Start connections/s test using 64 threads
ethr -c localhost -t c -n 64
```

## Complete Command Line
### Common Parameters
```
	-h 
		Help
	-no 
		Disable logging to file. Logging to file is enabled by default.
	-o <filename>
		Name of log file. By default, following file names are used:
		Server mode: 'ethrs.log'
		Client mode: 'ethrc.log'
		External server mode: 'ethrxs.log'
		External client mode: 'ethrxc.log'
	-debug 
		Enable debug information in logging output.
	-4 
		Use only IP v4 version
	-6 
		Use only IP v6 version
```
### Server Parameters
```
	-s 
		Run in server mode.
	-ui 
		Show output in text UI.
	-ports <k=v,...>
		Use custom port numbers instead of default ones.
		A comma separated list of key=value pair is used.
		Key specifies the protocol, and value specifies base port.
		Ports used for various tests are calculated from base port.
		Example: For TCP, Bw: 9999, CPS: 9998, PPS: 9997, Latency: 9996
		Control is used for control channel communication for ethr.
		Note: Same configuration must be used on both client & server.
		Default: 'control=8888,tcp=9999,udp=9999,http=9899,https=9799'
```
### Client Parameters
```
	-c <server>
		Run in client mode and connect to <server>.
		Server is specified using name, FQDN or IP address.
	-r 
		For Bandwidth tests, send data from server to client.
	-d <duration>
		Duration for the test (format: <num>[ms | s | m | h]
		0: Run forever
		Default: 10s
	-n <number>
		Number of Parallel Sessions (and Threads).
		0: Equal to number of CPUs
		Default: 1
	-ncs 
		No Connection Stats would be printed if this flag is specified.
		This is useful for running with large number of connections as
		specified by -n option.
	-l <length>
		Length of buffer to use (format: <num>[KB | MB | GB])
		Only valid for Bandwidth tests. Max 1GB.
		Default: 16KB
	-p <protocol>
		Protocol ("tcp", "udp", "http", "https", or "icmp")
		Default: tcp
	-ports <k=v,...>
		Use custom port numbers instead of default ones.
		A comma separated list of key=value pair is used.
		Key specifies the protocol, and value specifies base port.
		Ports used for various tests are calculated from base port.
		Example: For TCP, Bw: 9999, CPS: 9998, PPS: 9997, Latency: 9996
		Control is used for control channel communication for ethr.
		Note: Same configuration must be used on both client & server.
		Default: 'control=8888,tcp=9999,udp=9999,http=9899,https=9799'
	-t <test>
		Test to run ("b", "c", "p", or "l")
		b: Bandwidth
		c: Connections/s or Requests/s
		p: Packets/s
		l: Latency, Loss & Jitter
		Default: b - Bandwidth measurement.
	-i <iterations>
		Number of round trip iterations for each latency measurement.
		Default: 1000
```
### External Server Parameters
```
	-m <mode>
		'-m x' MUST be specified for external mode.
	-s 
		Run in server mode.
	-ports <k=v,...>
		Use custom port numbers instead of default ones.
		A comma separated list of key=value pair is used.
		Key specifies the protocol, and value specifies the port.
		Default: 'tcp=9999,http=9899,https=9799'
```
### External Client Mode
```
	-m <mode>
		'-m x' MUST be specified for external mode.
	-c <destination>
		Run in external client mode and connect to <destination>.
		<destination> is specified using host:port format.
		Example: www.microsoft.com:443 or 10.1.0.4:22 etc.
	-d <duration>
		Duration for the test (format: <num>[ms | s | m | h]
		0: Run forever
		Default: 10s
	-n <number>
		Number of Parallel Sessions (and Threads).
		0: Equal to number of CPUs
		Default: 1
	-ncs 
		No Connection Stats would be printed if this flag is specified.
		This is useful for running with large number of connections as
		specified by -n option.
	-l <length>
		Length of buffer to use (format: <num>[KB | MB | GB])
		Only valid for Bandwidth tests. Max 1GB.
		Default: 16KB
	-p <protocol>
		Protocol ("tcp", "http", "https", or "icmp")
		Default: tcp
	-t <test>
		Test to run ("b", "c", or "cl")
		b: Bandwidth
		c: Connections/s or Requests/s
		cl: TCP connection setup latency
		Default: b - Bandwidth measurement.
	-g <gap>
		Time interval between successive measurements (format: <num>[ms | s | m | h]
		0: No gap
		Default: 1s
```

# Status

Protocol  | Bandwidth | Connections/s | Packets/s | Latency
------------- | ------------- | ------------- | ------------- | -------------
TCP  | Yes | Yes | No | Yes
UDP  | Yes | NA | Yes | No
HTTP | Yes | No | No | Yes
HTTPS | Yes | No | No | No
ICMP | No | NA | No | No

# Platform Support

**Windows**

Tested: Windows 10, Windows 7 SP1

Untested: Other Windows versions

**Linux**

Tested: Ubuntu Linux 18.04.1 LTS, OpenSuse Leap 15

Untested: Other Linux versions

**OSX**

Tested: OSX is tested by contributors

**Other**

No other platforms are tested at this time

# Todo List

Todo list work items are shown below. Contributions are most welcome for these work items or any other features and bugfixes.

* Test Ethr on other Windows versions, other Linux versions, FreeBSD and other OS
* Support for UDP bandwidth & latency testing
* Support for HTTPS bandwidth, latency, requests/s
* Support for HTTP latency and requests/s
* Support for ICMP bandwidth, latency and packets/s

# Contributing

This project welcomes contributions and suggestions.  Most contributions require you to agree to a
Contributor License Agreement (CLA) declaring that you have the right to, and actually do, grant us
the rights to use your contribution. For details, visit https://cla.microsoft.com.

When you submit a pull request, a CLA-bot will automatically determine whether you need to provide
a CLA and decorate the PR appropriately (e.g., label, comment). Simply follow the instructions
provided by the bot. You will only need to do this once across all repos using our CLA.

This project has adopted the [Microsoft Open Source Code of Conduct](https://opensource.microsoft.com/codeofconduct/).
For more information see the [Code of Conduct FAQ](https://opensource.microsoft.com/codeofconduct/faq/) or
contact [opencode@microsoft.com](mailto:opencode@microsoft.com) with any additional questions or comments.
