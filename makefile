BINARY_NAME = golog
BUILD_DIR = bin
VERSION ?= $(shell git tag --sort=-creatordate | head -n 1)
OPTIONS = CGO_ENABLED=0
COMMIT = $(shell git rev-parse HEAD)
BUILD_TIME = $(shell date +%Y-%m-%dT%H:%M:%S)
BRANCH = $(shell git branch --show-current)

ENV = -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.Version=$(VERSION) -X main.Branch=$(BRANCH)
DARWIN_AMD = $(BUILD_DIR)/$(BINARY_NAME)_darwin_amd64
DARWIN_ARM = $(BUILD_DIR)/$(BINARY_NAME)_darwin_arm64
LINUX_AMD = $(BUILD_DIR)/$(BINARY_NAME)_linux_amd64
LINUX_ARM = $(BUILD_DIR)/$(BINARY_NAME)_linux_arm64
WIN_AMD = $(BUILD_DIR)/$(BINARY_NAME)_windows_amd64
WIN_ARM = $(BUILD_DIR)/$(BINARY_NAME)_windows_arm64

default: build

build:
	# Building  macOS  amd64
	GOOS=darwin GOARCH=amd64 $(OPTIONS) go build -trimpath -ldflags="-s -w $(ENV)" -o $(DARWIN_AMD)/$(BINARY_NAME) main.go

	# Building  macOS  arm64
	GOOS=darwin GOARCH=arm64 $(OPTIONS) go build -trimpath -ldflags="-s -w -w $(ENV)" -o $(DARWIN_ARM)/$(BINARY_NAME) main.go

	# Building  linux  amd64
	GOOS=linux GOARCH=amd64 $(OPTIONS) go build -trimpath -ldflags="-s -w $(ENV)" -o $(LINUX_AMD)/$(BINARY_NAME) main.go

	# Building  linux  arm64
	GOOS=linux GOARCH=arm64 $(OPTIONS)  go build -trimpath -ldflags="-s -w $(ENV)" -o $(LINUX_ARM)/$(BINARY_NAME)  main.go

	# Building  Windows  amd64
	GOOS=windows GOARCH=amd64 $(OPTIONS) go build -trimpath -ldflags="-s -w $(ENV)" -o $(WIN_AMD)/$(BINARY_NAME).exe main.go

	# Building  Windows  arm64
	GOOS=windows GOARCH=arm64 $(OPTIONS) go build -trimpath -ldflags="-s -w $(ENV)" -o $(WIN_ARM)/$(BINARY_NAME).exe main.go

clean:
	rm -rf $(BUILD_DIR)/$(BINARY_NAME)_*/$(BINARY_NAME)

.PHONY: build clean
