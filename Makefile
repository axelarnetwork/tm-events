PACKAGES=$(shell go list ./... | grep -v '/simulation')
VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

.PHONY: all
all: tmrpc

.PHONY: tmrpc
tmrpc: deps
	go build -o ./bin/tmrpc -mod=readonly ./cmd/tmrpc

.PHONY: install-tmrpc
install-tmrpc: deps
	go install ./cmd/tmrpc

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	GO111MODULE=on go mod verify

# Uncomment when you have some tests
# test:
# 	@go test -mod=readonly $(PACKAGES)
.PHONY:
	# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

# Run all the code generators in the project
.PHONY: generate
generate: prereqs
	go generate -x ./...

# Prepare go deps, as well as zeromq
.PHONY: deps
deps:
	@echo "--> Ensure build dependencies are present in the system"
	go mod tidy
	@echo 'checking zeromq dependencies'; sh -c 'pkg-config --modversion libzmq'

# Build a release image
.PHONY: docker-image
docker-image: deps
	@DOCKER_BUILDKIT=1 docker build --ssh default -t axelar/tmrpc .
