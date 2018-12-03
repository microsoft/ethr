# Ethr 

Ethr is a cross platform network performance measurement tool written in golang. Goal of this project is to provide common performance measurements such as bandwidth, connections/s, packets/s, latency, loss & jitter, across multiple protocols such as TCP, UDP, HTTP, HTTPS etc.

<p align="center">
  <img alt="Ethr server in action" src="https://user-images.githubusercontent.com/44273634/49360895-629cce80-f68f-11e8-967a-ed1f4c0ae6b6.png">
</p>

# Building

```bash
go build
```

# Usage

Help:
```bash
./ethr -h
```

Server:
```bash
./ethr -s
```

Server with Text UI:
```bash
./ethr -s -ui
```

Client:
```bash
./ethr -c <server ip>
```

# Status

Protocol  | Bandwidth | Connections/s | Packets/s | Latency
------------- | ------------- | ------------- | ------------- | -------------
TCP  | Yes | Yes | No | Yes
UDP  | No | NA | Yes | No
HTTP | Yes | No | No | No
HTTPS | No | No | No | No
ICMP | No | No | No | No

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
