.PHONY: build
build: deps
	go build -o ./bin/tmrpc -mod=readonly ./cmd

.PHONY: install
install: deps
	go install ./cmd

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	GO111MODULE=on go mod verify

# Uncomment when you have some tests
# test:
# 	@go test -mod=readonly $(PACKAGES)
.PHONY: lint
	# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify

# Run all the code generators in the project
.PHONY: generate
generate: prereqs
	go generate -x ./...

.PHONY: prereqs
prereqs:
ifeq (which moq,)
	go get -u github.com/matryer/moq
endif
ifeq (which mdformat,)
	pip3 install mdformat
endif
ifeq (which protoc,)
	@echo "Please install protoc for grpc (https://grpc.io/docs/languages/go/quickstart/)"
endif

# Prepare go deps, as well as zeromq
.PHONY: deps
deps: go.sum
	@echo "--> Ensure build dependencies are present in the system"
	go mod tidy
	@echo 'checking zeromq dependencies'; sh -c 'pkg-config --modversion libzmq'

# Build a release image
.PHONY: docker-image
docker-image: deps
	@DOCKER_BUILDKIT=1 docker build --ssh default -t axelar/tmrpc .
