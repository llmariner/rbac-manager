.PHONY: default
default: test

include common.mk

.PHONY: test
test: go-test-all

.PHONY: lint
lint: go-lint-all helm-lint git-clean-check

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

.PHONY: check-helm-tool
check-helm-tool:
	@command -v helm-tool >/dev/null 2>&1 || $(MAKE) install-helm-tool

.PHONY: install-helm-tool
install-helm-tool:
	go install github.com/cert-manager/helm-tool@latest

.PHONY: generate-chart-schema
generate-chart-schema: generate-chart-schema-dex generate-chart-schema-rbac

.PHONY: generate-chart-schema-dex
generate-chart-schema-dex: check-helm-tool
	@cd ./deployments/dex-server && helm-tool schema > values.schema.json

.PHONY: generate-chart-schema-rbac
generate-chart-schema-rbac: check-helm-tool
	@cd ./deployments/rbac-server && helm-tool schema > values.schema.json

.PHONY: helm-lint
helm-lint: helm-lint-dex helm-lint-rbac

.PHONY: helm-lint-dex
helm-lint-dex: generate-chart-schema-dex
	cd ./deployments/dex-server && helm-tool lint
	helm lint ./deployments/dex-server

.PHONY: helm-lint-rbac
helm-lint-rbac: generate-chart-schema-rbac
	cd ./deployments/rbac-server && helm-tool lint
	helm lint ./deployments/rbac-server
