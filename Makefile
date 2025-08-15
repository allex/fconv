## Makefile for build golang binary
# by allex_wang

SHELL := $(shell which bash)

# Packages in vendor/ are included in ./...
# https://github.com/golang/go/issues/11659
OUR_PACKAGES=$(shell go list ./... | grep -v '/vendor/')
GOX = gox
BUILT := $(shell date -u +%Y-%m-%dT%H:%M:%S%z)

GIT_COMMIT := $(shell git rev-parse --short HEAD)
LATEST_TAG := $(shell git log --decorate --pretty="format:%d" | awk 'match($$0, "[(]?tag:\\s*v?([^,]+?)[,)]", arr) { if(arr[1] ~ "^.+?[0-9]+\\.[0-9]+\\.[0-9]+(-.+)?$$") print arr[1]; exit; }')

# flag to disable automate version if set true
STATIC_VERSION := false

# prerelease tag, such as dev,rc,next etc,.
prerelease ?=

# Specify the release type manully, <major|minor|patch>, default release as last tag
# increase patch version when prerelease mode
release_as ?= patch

# check static version mode, set with args: release_as=x.y.z
ifeq ($(shell echo "$(release_as)" | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+(-.+)?$$'),$(release_as))
	# release_as is set to a valid version value
	STATIC_VERSION := true
	LATEST_TAG := $(release_as)
endif

IS_LATEST := false
ifeq ($(shell git describe --tags --exact-match --match $(LATEST_TAG) >/dev/null 2>&1; echo $$?), 0)
	STATIC_VERSION := true
	IS_LATEST := true
endif

# set prerelease to dev if not static releasing
ifneq ($(STATIC_VERSION),true)
	ifneq ($(origin prerelease), command line)
		prerelease := dev
	endif
endif

get_version = \
	set -eu; \
	ver=$(LATEST_TAG); \
	[ -n "$$ver" ] || exit 1; \
	if [ "$(IS_LATEST)" != "true" -a $(STATIC_VERSION) != "true" ]; then \
		release_as=$$(echo $(release_as) | sed "s/major/M/;s/minor/m/;s/patch/p/"); \
		ver=$$(echo "$$ver" | awk -v release_as=$$release_as 'BEGIN{FS=OFS="."} release_as~"^v?[0-9]+(\\.[0-9]+)*$$"{print gensub("^v","","g",release_as);exit} $$0~"(\\.[0-9]+)+$$"{ i=index("Mmp", release_as); if (i!=0) { $$i++; while (i<3) {$$(++i)=0} } print }'); \
		ver=$${ver:-$(LATEST_TAG)}; \
	fi; \
	prerelease=$(prerelease); \
	if [ -n "$$prerelease" ]; then \
		ver="$${ver%%-$$prerelease}-$$prerelease"; \
	fi; \
	echo $$ver

release_tag := $(shell $(get_version))

.PHONY: version build release clean help test

# Parse Makefile and display the help
## > help - Show help
help:
	# Commands:
	@grep -E '^## > [a-zA-Z_-]+.*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = "## > "}; {printf ">\033[36m%-1s\033[0m %s\n", $$1, $$2}'

## > release - Update and commit release
release:
	git release -t $(argument)

## > version [release_as=patch|minor|major|x.y.z] [prerelease=dev|rc|xxx] - Show versions info
version:
	@printf "Current version: %s (commit: %s, is_latest: %s)\n" $(LATEST_TAG) $(GIT_COMMIT) $(IS_LATEST)
	@printf "Version: %s\n" $(release_tag)

## > build [release_as=patch|minor|major|x.y.z] [prerelease=dev|rc|xxx] - build project for all support OSes
build: clean
	# $(GOX) -os "darwin linux windows" -arch="amd64 arm64" -output "out/fconv-{{.OS}}-{{.Arch}}" ./
	@set -x;\
	declare -A platform=( \
		[darwin]="386 amd64" \
		[linux]="386 amd64 arm64" \
	); \
	for GOOS in $${!platform[@]}; do \
		for GOARCH in $${platform[$$GOOS]}; do \
			(\
				export GOOS GOARCH; \
				o=$$PWD/out/fconv-$$GOOS-$$GOARCH; \
				mkdir -p $$(dirname $$o); \
				go build -ldflags "-w -s -X main.appVersion=v$(release_tag) -X main.gitCommit=$(GIT_COMMIT) -X 'main.buildTime=$(BUILT)'" -v -o $$o; \
				if [ $$GOOS = "darwin" ]; then upx $$o &>/dev/null; fi \
			) \
		done \
	done

## > publish [PREFIX=tdio][release_as][prerelease] - publish docker image
publish:
	export BUILD_TAG=$(release_tag); \
	docker buildx bake \
		--set "main.args.GIT_COMMIT=$(GIT_COMMIT)" \
		--set "main.args.BUILD_TIME=$(BUILT)" \
		--push

clean:
	rm -rf out

## > test [UPDATE_SNAPSHOTS=true] - run project tests
test:
	# Running tests...
	@go test $(OUR_PACKAGES) -cover -v
