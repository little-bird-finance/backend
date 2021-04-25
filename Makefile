include .env

.DEFAULT_GOAL := all

.PHONY: help run test cover test-integration test-unit docker-integration-start docker-integration-stop watch-test

# PROTOS_SRC          := $(wildcard entity/*.proto)
# PROTOS_BIN          := $(PROTOS_SRC:.proto=.pb.go)
BIN_PATH             := dist
COVER_FILE_PATH      := $(BIN_PATH)/coverage.out
SCANAPI_REPORT_PATH  := $(BIN_PATH)/scanapi-report.html
DOC_ADDR             := :8081
TESTS_PATH           := ./...
CONTAINER_CMD        := docker
CONTAINER_IMAGE_NAME := little-bird-finance-backend
CONTAINER_IMAGE_TAG  := latest

all: cover-file test-unit build

help:  ## Show help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# protobuf: $(PROTOS_BIN) ## make protobuf files 

# $(PROTOS_BIN): $(PROTOS_SRC)
# 	protoc --experimental_allow_proto3_optional -I=entity --go_out=entity $(@:.pb.go=.proto)

clean: ## Clean up build files
	rm -rf dist
	#rm -f $(PROTOS_BIN)

$(BIN_PATH):
	mkdir -p $(BIN_PATH)

$(COVER_FILE_PATH): $(BIN_PATH)
	go test -coverprofile=$(COVER_FILE_PATH) $(TESTS_PATH)

$(SCANAPI_REPORT_PATH): $(BIN_PATH)
	scanapi run -o $(SCANAPI_REPORT_PATH) ops/scanapi/scanapi.yml

cover:  ## Run coverage tests
	go test -cover $(TESTS_PATH)
	
cover-func: $(COVER_FILE_PATH)  ## Run coverage tests by function
	go tool cover -func=$(COVER_FILE_PATH)

cover-file: $(COVER_FILE_PATH) ## Create a file with coverage test

cover-browser: cover-file ## Show coverage test in a browser
	go tool cover -html=$(COVER_FILE_PATH)

test-unit: ## Run all unit tests
	go test $(TESTS_PATH)

# test-integration: ## 
# 	go test -tags integration ./...

test-scanapi: $(SCANAPI_REPORT_PATH) ## Run all integration tests with scanapi

run: ## Start a http server
	go run cmd/main.go

doc: ## Start a go doc server, need to have installed go tools: go get -u golang.org/x/tools/...
	godoc -http $(DOC_ADDR)

build: $(BIN_PATH) ## Create a binary
	go build -o $(BIN_PATH)/backend cmd/main.go

build-all: $(BIN_PATH) ## Create a binary for each platform
	echo "Compiling for every OS and Platform"
	GOOS=linux GOARCH=arm go build -o $(BIN_PATH)/backend-linux-arm cmd/main.go
	GOOS=linux GOARCH=arm64 go build -o $(BIN_PATH)/backend-linux-arm64 cmd/main.go
	GOOS=linux GOARCH=amd64 go build -o $(BIN_PATH)/backend-linux-amd64 cmd/main.go
	GOOS=windows GOARCH=amd64 go build -o $(BIN_PATH)/backend-windows-amd64 cmd/main.go

# docker: ## Cria uma imagem docker
# 	docker build .
#
container-build: ## Create a container image
	$(CONTAINER_CMD) build -f ops/Containerfile -t $(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG) .
# #go test -run=.*/trailing -v
#
container-run: ## Create a container image
	$(CONTAINER_CMD) run -p 3000:3000 --env-file .env $(CONTAINER_IMAGE_NAME):$(CONTAINER_IMAGE_TAG)
# #go test -run=.*/trailing -v

fmt: ## Format all code with gofmt
	gofmt -s -w .

# Integration tests create bin and not run
# go test -tags integration -c ./testing/integration/...

#build: ## Build project
#	go build -o bin/main .
#
#run: ## Run the code
#	go run cmd/main.go
#
#doc:
#	godoc -http :8080
#
#bench:
#	go test -bench .
#
#go-help:
#	go help testflag
#
#test: test-unit ## Run unit-test
#
#test-unit: ## Execute unit tests
#	go test ./...
#
#test-integration: docker-integration-start ## Run integration tests
#	SQL_CONNECTION_STRING="postgres://postgres:postgres@localhost/little-bird-finance-test?sslmode=disable"
#	-go test -tags integration
#	make docker-integration-stop
#
#docker-integration-start: ## Start docker for integrations tests
#	docker-compose -f docker-compose-integration.yml up -V -d
#
#docker-integration-stop: ## Stop docker of integration tests
#	docker-compose -f docker-compose-integration.yml down
#
#clean: ## Clean up build files
#	rm -rf bin
#
#bin/coverage.out:
#	mkdir -p bin
#	go test -coverprofile=bin/coverage.out ./...
#
#bin/coverage.html: bin/coverage.out
#	go tool cover -html=bin/coverage.out -o bin/coverage.html
#
#
#cover:  ## Run cover test and show
#	go test -cover ./...
#	
#cover-func: bin/coverage.out  ## Show coverage by function
#	go tool cover -func=bin/coverage.out
#
#cover-browser: bin/coverage.out  ## Show coverage report on browser
#	# go tool cover -html=bin/coverage.out
#	go tool cover -html=bin/coverage.out
#cover-html: bin/coverage.out  ## Create coverage on html
#	go tool cover -html=bin/coverage.out -o bin/coverage.html
#
#watch-test: ## Run tests every time any go files has modified
#	find . -name "*.go" | entr -cr make test
#
#protobuf: ## Generate protobuf files
#	protoc --experimental_allow_proto3_optional -I=entity --go_out=entity entity/expense.proto
