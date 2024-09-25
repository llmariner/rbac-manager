.PHONY: default
default: test

include common.mk

.PHONY: test
test: go-test-all

.PHONY: lint
lint: go-lint-all git-clean-check

.PHONY: generate
generate: buf-generate-all typescript-compile

.PHONY: build-server
build-server:
	go build -o ./bin/server ./server/cmd/

.PHONY: build-database-creator
build-database-creator:
	go build -o ./bin/database-creator ./database-creator/cmd/

.PHONY: build-docker-server
build-docker-server:
	docker build --build-arg TARGETARCH=amd64 -t llmariner/rbac-server:latest -f build/server/Dockerfile .

.PHONY: build-docker-database-creator
build-docker-database-creator:
	docker build --build-arg TARGETARCH=amd64 -t llmariner/database-creator:latest -f build/database-creator/Dockerfile .

.PHONY: build-docker-envsubst
build-docker-envsubst:
	docker build --build-arg TARGETARCH=amd64 -t llmariner/envsubst:latest -f build/envsubst/Dockerfile .
