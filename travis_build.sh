#!/bin/bash
echo "${TRAVIS_OS_NAME}"
echo "${TRAVIS_GO_VERSION}"
if [ "${TRAVIS_OS_NAME}" = "linux" ]; then
    export GOOS=windows
    export GOARCH=amd64
    go build -o windows/ethr.exe -ldflags "-X main.gVersion=$TRAVIS_TAG"
    export GOOS=linux
    go build -o linux/ethr -ldflags "-X main.gVersion=$TRAVIS_TAG"
    export GOOS=darwin
    go build -o osx/ethr -ldflags "-X main.gVersion=$TRAVIS_TAG"
    zip -j ethr_windows.zip windows/ethr.exe
    zip -j ethr_linux.zip linux/ethr
    zip -j ethr_osx.zip osx/ethr
fi

