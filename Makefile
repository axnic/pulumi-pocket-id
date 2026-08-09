PROJECT_NAME := Pulumi Pocket-ID Provider

PACK             := pocket-id
PACKDIR          := sdk
PROJECT          := github.com/axnic/pulumi-pocket-id
NODE_MODULE_NAME := @pulumi/pocket-id
NUGET_PKG_NAME   := Pulumi.PocketId

PROVIDER        := pulumi-resource-${PACK}
PROVIDER_PATH   := provider
VERSION_PATH    := ${PROVIDER_PATH}.Version

PULUMI          := pulumi

SCHEMA_FILE     := provider/cmd/pulumi-resource-pocket-id/schema.json
export GOPATH   := $(shell go env GOPATH)

WORKING_DIR     := $(shell pwd)
TESTPARALLELISM := 4

prepare:
	@if test -z "${NAME}"; then echo "NAME not set"; exit 1; fi
	@if test -z "${REPOSITORY}"; then echo "REPOSITORY not set"; exit 1; fi
	@if test -z "${ORG}"; then echo "ORG not set"; exit 1; fi
	@if test ! -d "provider/cmd/pulumi-resource-pocket-id"; then "Project already prepared"; exit 1; fi # SED_SKIP

	# SED needs to not fail when encountering unicode characters
	LC_CTYPE=C 
	LANG=C

	mv "provider/cmd/pulumi-resource-pocket-id" provider/cmd/pulumi-resource-${NAME} # SED_SKIP
	
	# In MacOS the -i parameter needs an empty  to execute in place.
	if [[ "${OS}" == "Darwin" ]]; then \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s,github.com/pulumi/pulumi-[x]yz,${REPOSITORY},g' {} \; ; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s/[xX]yz/${NAME}/g' {} \; ; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '' '/SED_SKIP/!s/[aA]bc/${ORG}/g' {} \; ; \
	else \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s,github.com/pulumi/pulumi-[x]yz,${REPOSITORY},g' {} \; ; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s/[xX]yz/${NAME}/g' {} \; ; \
		find . \( -path './.git' -o -path './sdk' \) -prune -o -not -name 'go.sum' -type f -exec sed -i '/SED_SKIP/!s/[aA]bc/${ORG}/g' {} \; ; \
	fi

# Override during CI using `make [TARGET] PROVIDER_VERSION=""` or by setting a PROVIDER_VERSION environment variable
# Local & branch builds will just used this fixed default version unless specified
PROVIDER_VERSION ?= 1.0.0-alpha.0+dev
# Use this normalised version everywhere rather than the raw input to ensure consistency.
VERSION_GENERIC = $(shell pulumictl convert-version --language generic --version "$(PROVIDER_VERSION)")

# Need to pick up locally pinned pulumi-langage-* plugins.
export PULUMI_IGNORE_AMBIENT_PLUGINS = true

ensure::
	go mod tidy

$(SCHEMA_FILE): provider
	$(PULUMI) package get-schema $(WORKING_DIR)/bin/${PROVIDER} | \
		jq 'del(.version)' > $(SCHEMA_FILE)

# Codegen generates the schema file and *generates* all sdks. This is a local process and
# does not require the ability to build all SDKs.
#
# To build the SDKs, use `make build_sdks`
codegen: $(SCHEMA_FILE) sdk/dotnet sdk/go sdk/nodejs sdk/python

.PHONY: sdk/%
sdk/%: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language $* $(SCHEMA_FILE) --version "${VERSION_GENERIC}"

sdk/python: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language python $(SCHEMA_FILE) --version "${VERSION_GENERIC}"
	cp README.md ${PACKDIR}/python/

sdk/dotnet: $(SCHEMA_FILE)
	rm -rf $@
	$(PULUMI) package gen-sdk --language dotnet $(SCHEMA_FILE) --version "${VERSION_GENERIC}"


sdk/go: ${SCHEMA_FILE}
	rm -rf $@
	$(PULUMI) package gen-sdk --language go ${SCHEMA_FILE} --version "${VERSION_GENERIC}"
	cp go.mod ${PACKDIR}/go/pulumi-${PACK}/go.mod
	cd ${PACKDIR}/go/pulumi-${PACK} && \
		go mod edit -module=${PROJECT}/${PACKDIR}/go/pulumi-${PACK} && \
		go mod tidy

.PHONY: provider
provider: bin/${PROVIDER} bin/pulumi-gen-${PACK} # Required by CI

bin/${PROVIDER}:
	cd provider && go build -o $(WORKING_DIR)/bin/${PROVIDER} -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION_GENERIC}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER)

.PHONY: provider_debug
provider_debug:
	(cd provider && go build -o $(WORKING_DIR)/bin/${PROVIDER} -gcflags="all=-N -l" -ldflags "-X ${PROJECT}/${VERSION_PATH}=${VERSION_GENERIC}" $(PROJECT)/${PROVIDER_PATH}/cmd/$(PROVIDER))

test_provider:
	cd provider && go test -short -v -count=1 -cover -timeout 2h -parallel ${TESTPARALLELISM} -coverprofile="coverage.txt" ./...

dotnet_sdk: sdk/dotnet
	cd ${PACKDIR}/dotnet/&& \
		echo "${VERSION_GENERIC}" > version.txt && \
		dotnet build

go_sdk:	sdk/go

nodejs_sdk: sdk/nodejs
	cd ${PACKDIR}/nodejs/ && \
		yarn install && \
		yarn run tsc
	cp README.md LICENSE ${PACKDIR}/nodejs/package.json ${PACKDIR}/nodejs/yarn.lock ${PACKDIR}/nodejs/bin/

python_sdk: sdk/python
	cp README.md ${PACKDIR}/python/
	cd ${PACKDIR}/python/ && \
		rm -rf ./bin/ ../python.bin/ && cp -R . ../python.bin && mv ../python.bin ./bin && \
		python3 -m venv venv && \
		./venv/bin/python -m pip install build && \
		cd ./bin && \
		../venv/bin/python -m build .

.PHONY: build
build:: provider build_sdks

.PHONY: build_sdks
build_sdks: dotnet_sdk go_sdk nodejs_sdk python_sdk

# Required for the codegen action that runs in pulumi/pulumi
only_build:: build

lint:
	golangci-lint --path-prefix provider --config .golangci.yml run --fix


install:: install_nodejs_sdk install_dotnet_sdk
	cp $(WORKING_DIR)/bin/${PROVIDER} ${GOPATH}/bin


install_dotnet_sdk::
	rm -rf $(WORKING_DIR)/nuget/$(NUGET_PKG_NAME).*.nupkg
	mkdir -p $(WORKING_DIR)/nuget
	find . -name '*.nupkg' -print -exec cp -p {} ${WORKING_DIR}/nuget \;

install_python_sdk::
	#target intentionally blank

install_go_sdk::
	#target intentionally blank

install_nodejs_sdk::
	-yarn unlink --cwd $(WORKING_DIR)/sdk/nodejs/bin
	yarn link --cwd $(WORKING_DIR)/sdk/nodejs/bin

test:: test_provider
	cd examples && go test -v -tags=all -timeout 2h

# test_e2e starts a real, disposable Pocket-ID instance via Docker and runs
# the acceptance suite against it. Requires Docker; see
# docker-compose.test.yml. Pin a Pocket-ID version with POCKET_ID_IMAGE, e.g.
# `POCKET_ID_IMAGE=pocketid/pocket-id:v2.9.0 make test_e2e` - STATIC_API_KEY
# support requires Pocket-ID >= v2.3.0.
.PHONY: test_e2e
test_e2e:
	docker compose -f docker-compose.test.yml up --detach --wait
	POCKET_ID_BASE_URL=http://localhost:1411 POCKET_ID_API_KEY=pulumi-test-api-key-0123456789 \
		go test ./tests/... -run TestAcceptance -count=1 -v; \
		status=$$?; \
		docker compose -f docker-compose.test.yml down --volumes; \
		exit $$status

# dev-up/dev-down start and stop the same Pocket-ID instance for interactive
# local development (no automated teardown - it's left running for you to
# point `pulumi up`, curl, or the examples at). See CONTRIBUTING.md's Setting
# up your development environment section.
.PHONY: dev-up
dev-up:
	docker compose -f docker-compose.test.yml up --detach --wait
	@echo "Pocket-ID:          http://localhost:1411"
	@echo "export POCKET_ID_BASE_URL=http://localhost:1411"
	@echo "export POCKET_ID_API_KEY=pulumi-test-api-key-0123456789"

.PHONY: dev-down
dev-down:
	docker compose -f docker-compose.test.yml down --volumes

.PHONY:local_generate
local_generate: # Required by CI

.PHONY: generate_schema
generate_schema: ${SCHEMA_PATH} # Required by CI

.PHONY: build_go install_go_sdk
generate_go: sdk/go # Required by CI
build_go: # Required by CI

.PHONY: build_python install_python_sdk
generate_python: sdk/python # Required by CI
build_python: python_sdk # Required by CI

.PHONY: build_nodejs install_nodejs_sdk
generate_nodejs: sdk/nodejs # Required by CI
build_nodejs: nodejs_sdk # Required by CI

.PHONY: build_dotnet install_dotnet_sdk
generate_dotnet: sdk/dotnet # Required by CI
build_dotnet: dotnet_sdk # Required by CI

bin/pulumi-gen-${PACK}: # Required by CI
	touch bin/pulumi-gen-${PACK}
