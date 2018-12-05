# Ethr 

Ethr is a cross platform network performance measurement tool written in golang. Goal of this project is to provide native tool for network performance measurements for bandwidth, connections/s, packets/s, latency, loss & jitter, across multiple protocols such as TCP, UDP, HTTP, HTTPS, and across multiple platforms such as Windows, Linux and other Unix systems.

<p align="center">
  <img alt="Ethr server in action" src="https://user-images.githubusercontent.com/44273634/49360895-629cce80-f68f-11e8-967a-ed1f4c0ae6b6.png">
</p>

Ethr takes insipiration from existing open source network performance tools and builds upon those ideas. It is very similar to iPerf3 for bandwidth measurements for TCP. iPerf3 has many more options for doing bandwidth measurements such as throttled testing, richer feature set, while Ethr has support for multiple threads, ability to scale to 1024 or even higher connections, multiple clients to single server etc. It is similar to latte on Windows or sockperf on Linux for doing latency measurements.

Ethr is natively cross platform, thanks to golang, as compared to compiling via abstraction layer like cygwin that may limit functionality. It hopes to unify performance measurement by combining functionality of tools like iPerf3, ntttcp, psping, sockperf, latte, and many other available today on different platforms and offer a single tool across multiple platforms.

Ethr provides more test measurements as compared to other tools, e.g. it provides measurements for connections/s, packets/s and latency, all in a single tool. In future, there is plan to add more features (hoping for others to contribute) as well as more protocol support to make it a comprehensive tool for network performance measurements.

# Download

```bash
For Windows 10: https://github.com/Microsoft/Ethr/files/2640289/ethr.zip
For Ubuntu: https://github.com/Microsoft/Ethr/files/2640288/ethr.gz
```

# Installation

Note: go version 1.10 or higher is required building it from the source.

## Building from Source

```bash
git clone https://github.com/Microsoft/ethr.git
cd ethr
dep ensure -v
go build
```

## Using go get

```bash
go get github.com/Microsoft/ethr
```

# Usage

Help:
```bash
ethr -h
```

Server:
```bash
ethr -s
```

Server with Text UI:
```bash
ethr -s -ui
```

Client:
```bash
ethr -c <server ip>
```

Example:
```bash
// Start server
ethr -s

// Start client for default (bandwidth) test measurement using 1 thread
ethr -c localhost

// Start connections/s test using 64 threads
ethr -c localhost -t c -n 64 
```

# Status

Protocol  | Bandwidth | Connections/s | Packets/s | Latency
------------- | ------------- | ------------- | ------------- | -------------
TCP  | Yes | Yes | No | Yes
UDP  | No | NA | Yes | No
HTTP | Yes | No | No | No
HTTPS | No | No | No | No
ICMP | No | NA | No | No

# Platform Support

**Windows**

Tested: Windows 10

Untested: Other Windows versions

**Linux**

Tested: Ubuntu Linux 18.04.1 LTS

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
