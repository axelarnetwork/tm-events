.PHONY: build
build: deps
	go build -o ./bin/tmrpc -mod=readonly ./cmd/tmrpc

.PHONY: install
install: deps
	go install ./cmd/tmrpc

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
generate: go.sum prereqs goimports
	go generate ./...

.PHONY: goimports
goimports:
	@echo "running goimports"
# exclude mocks and proto generated files
	@goimports -l -local github.com/axelarnetwork/ . | grep -v .pb.go$ | grep -v mock | xargs goimports -local github.com/axelarnetwork/ -w

# Install all generate prerequisites
.Phony: prereqs
prereqs:
	@which goimports &>/dev/null	||	go install golang.org/x/tools/cmd/goimports
	@which moq &>/dev/null			||	go install github.com/matryer/moq
	@which mdformat &>/dev/null 	||	pip3 install mdformat
	@which protoc &>/dev/null 		|| 	echo "Please install protoc for grpc (https://grpc.io/docs/languages/go/quickstart/)"

# Prepare go deps, as well as zeromq
.PHONY: deps
deps: go.sum
	@echo "--> Ensure build dependencies are present in the system"
	go mod tidy
	@echo 'checking zeromq dependencies'; sh -c 'pkg-config --modversion libzmq'

# Build a release image
.PHONY: docker-image
docker-image:
	@DOCKER_BUILDKIT=1 docker build -t axelar/tmrpc .
