# Ethr

Ethr is a cross platform network performance measurement tool written in golang. Goal of this project is to provide a native tool for network performance measurements of bandwidth, connections/s, packets/s, latency, loss & jitter, across multiple protocols such as TCP, UDP, HTTP, HTTPS, and across multiple platforms such as Windows, Linux and other Unix systems.

<p align="center">
  <img alt="Ethr server in action" src="https://user-images.githubusercontent.com/44273634/49815752-506f0000-fd21-11e8-954e-d587e79c5d85.png">
</p>

Ethr takes insipiration from existing open source network performance tools and builds upon those ideas. It is very similar to iPerf3 for bandwidth measurements for TCP. iPerf3 has many more options for doing bandwidth measurements such as throttled testing, richer feature set, while Ethr has support for multiple threads, ability to scale to 1024 or even higher connections, multiple clients to single server etc. It is similar to latte on Windows or sockperf on Linux for doing latency measurements.

Ethr is natively cross platform, thanks to golang, as compared to compiling via abstraction layer like cygwin that may limit functionality. It hopes to unify performance measurement by combining functionality of tools like iPerf3, ntttcp, psping, sockperf, latte, and many other available today on different platforms and offer a single tool across multiple platforms.

Ethr provides more test measurements as compared to other tools, e.g. it provides measurements for connections/s, packets/s and latency, all in a single tool. In future, there is plan to add more features (hoping for others to contribute) as well as more protocol support to make it a comprehensive tool for network performance measurements.

# Download

```
For Windows 10: https://github.com/Microsoft/Ethr/files/2640289/ethr.zip
For Ubuntu: https://github.com/Microsoft/Ethr/files/2640288/ethr.gz
For ArchLinux: https://aur.archlinux.org/packages/ethr
```

# Installation

Note: go version 1.10 or higher is required building it from the source.

## Building from Source

```
git clone https://github.com/Microsoft/ethr.git
cd ethr
dep ensure -v
go build
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

Example:
```
// Start server
ethr -s

// Start client for default (bandwidth) test measurement using 1 thread
ethr -c localhost

// Start connections/s test using 64 threads
ethr -c localhost -t c -n 64
```

## Complete Command Line
### Common Parameters
```
-h                        Help
-no                       Disable logging to a file
-o <filename>             Log to the file specified by filename. 
                          By default Ethr logs to ./ethrs.log for server & ./ethrc.log for client mode
-debug                    Log debug output
```
### Server Parameters
```
-s                        Server mode
-ui                       Display text UI
```
### Client Parameters
```
-c <server>                   Client mode, connect to name or IP specified by server
-t <b|c|p|l>                  Test to be done, b: bandwidth, c: connections/s, p: packets/s, l: latency
                              Default is bandwidth test
-p <tcp|udp|http|https|icmp>  Protocol to use, default is TCP
-n <number>                   Number of sessions/threads to use
-l <number>                   Buffer size to use for each request
-i <number>                   Number of iterations for latency test
```

# Status

Protocol  | Bandwidth | Connections/s | Packets/s | Latency
------------- | ------------- | ------------- | ------------- | -------------
TCP  | Yes | Yes | No | Yes
UDP  | Yes | NA | Yes | No
HTTP | Yes | No | No | No
HTTPS | No | No | No | No
ICMP | No | NA | No | No

# Platform Support

**Windows**

Tested: Windows 10, Windows 7 SP1

Untested: Other Windows versions

**Linux**

Tested: Ubuntu Linux 18.04.1 LTS, OpenSuse Leap 15

Untested: Other Linux versions

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
