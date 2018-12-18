#!make
#-----------------------------------------------------------------------------
# Copyright (C) Microsoft. All rights reserved.
# Licensed under the MIT license.
# See LICENSE.txt file in the project root for full license information.
#-----------------------------------------------------------------------------

GO_BIN ?= go
GO_LINT ?= golint
GO_FMT ?= gofmt
BINARY_NAME ?= ethr

.PHONY: fmt
fmt:
	find . -name '*.go' | \
	    while read -r file; \
	        do $(GO_FMT) -w -s "$$file"; \
	    done

.PHONY: build-docker
build-docker: 
	$(GO_BIN) build -o /out/$(BINARY_NAME)

.PHONY: build
build:
	$(GO_BIN) build -o $(BINARY_NAME) .

.PHONY: lint
lint:
	$(GO_LINT) .

.DEFAULT_GOAL := build
