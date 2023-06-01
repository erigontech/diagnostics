BINARY_NAME := diagnostics
BUILD_DIR := ./_bin
DOCKER_IMAGE_NAME := diagnostics
DOCKER_CONTAINER_NAME := diagnostics_container

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/diagnostics

test:
	go test ./...

run:
	go run ./cmd/diagnostics/main.go

clean:
	rm -rf $(BUILD_DIR)

build-docker:
	docker build -t $(DOCKER_IMAGE_NAME) .

run-docker:
	docker run -p 8080:8080 --name $(DOCKER_CONTAINER_NAME) $(DOCKER_IMAGE_NAME)

## lint:                              run golangci-lint with .golangci.yml config file
lint:
	@./build/bin/golangci-lint run --config ./.golangci.yml

## lintci:                            run golangci-lint (additionally outputs message before run)
lintci:
	@echo "--> Running linter for code"
	@./build/bin/golangci-lint run --config ./.golangci.yml

## lintci-deps:                       (re)installs golangci-lint to build/bin/golangci-lint
lintci-deps:
	rm -f ./build/bin/golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./build/bin v1.52.2

.PHONY: build test run clean