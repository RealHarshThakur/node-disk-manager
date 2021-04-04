<<<<<<< HEAD
# Copyright 2018-2020 The OpenEBS Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Default behaviour is not to use BUILDX until the Travis workflow is deprecated.
BUILDX:=false

# ==============================================================================
# Build Options

# set the shell to bash in case some environments use sh
SHELL:=/bin/bash

# VERSION is the version of the binary.
VERSION:=$(shell git describe --tags --always)

# Determine the arch/os
ifeq (${XC_OS}, )
  XC_OS:=$(shell go env GOOS)
endif
export XC_OS

ifeq (${XC_ARCH}, )
  XC_ARCH:=$(shell go env GOARCH)
endif
export XC_ARCH

ARCH:=${XC_OS}_${XC_ARCH}
export ARCH

ifeq (${BASE_DOCKER_IMAGEARM64}, )
  BASE_DOCKER_IMAGEARM64 = "arm64v8/ubuntu:18.04"
  export BASE_DOCKER_IMAGEARM64
endif

ifeq (${BASEIMAGE}, )
ifeq ($(ARCH),linux_arm64)
  BASEIMAGE:=${BASE_DOCKER_IMAGEARM64}
else
  # The ubuntu:16.04 image is being used as base image.
  BASEIMAGE:=ubuntu:16.04
endif
endif
export BASEIMAGE

# The images can be pushed to any docker/image registeries
# like docker hub, quay. The registries are specified in
# the `build/push` script.
#
# The images of a project or company can then be grouped
# or hosted under a unique organization key like `openebs`
#
# Each component (container) will be pushed to a unique
# repository under an organization.
# Putting all this together, an unique uri for a given
# image comprises of:
#   <registry url>/<image org>/<image repo>:<image-tag>
#
# IMAGE_ORG can be used to customize the organization
# under which images should be pushed.
# By default the organization name is `openebs`.

ifeq (${IMAGE_ORG}, )
  IMAGE_ORG="openebs"
  export IMAGE_ORG
endif

# Specify the date of build
DBUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

# Specify the docker arg for repository url
ifeq (${DBUILD_REPO_URL}, )
  DBUILD_REPO_URL="https://github.com/openebs/node-disk-manager"
  export DBUILD_REPO_URL
endif

# Specify the docker arg for website url
ifeq (${DBUILD_SITE_URL}, )
  DBUILD_SITE_URL="https://openebs.io"
  export DBUILD_SITE_URL
endif

export DBUILD_ARGS=--build-arg DBUILD_DATE=${DBUILD_DATE} --build-arg DBUILD_REPO_URL=${DBUILD_REPO_URL} --build-arg DBUILD_SITE_URL=${DBUILD_SITE_URL} --build-arg RELEASE_TAG=${RELEASE_TAG}

# Initialize the NDM DaemonSet variables
# Specify the NDM DaemonSet binary name
NODE_DISK_MANAGER=ndm
# Specify the sub path under ./cmd/ for NDM DaemonSet
BUILD_PATH_NDM=ndm_daemonset
# Name of the image for NDM DaemoneSet
DOCKER_IMAGE_NDM:=${IMAGE_ORG}/node-disk-manager-${XC_ARCH}:ci

# Initialize the NDM Operator variables
# Specify the NDM Operator binary name
NODE_DISK_OPERATOR=ndo
# Specify the sub path under ./cmd/ for NDM Operator
BUILD_PATH_NDO=manager
# Name of the image for ndm operator
DOCKER_IMAGE_NDO:=${IMAGE_ORG}/node-disk-operator-${XC_ARCH}:ci

# Initialize the NDM Exporter variables
# Specfiy the NDM Exporter binary name
NODE_DISK_EXPORTER=exporter
# Specify the sub path under ./cmd/ for NDM Exporter
BUILD_PATH_EXPORTER=ndm-exporter
# Name of the image for ndm exporter
DOCKER_IMAGE_EXPORTER:=${IMAGE_ORG}/node-disk-exporter-${XC_ARCH}:ci

# Compile binaries and build docker images
.PHONY: build
build: clean build.common docker.ndm docker.ndo docker.exporter

.PHONY: build.common
build.common: license-check version

# If there are any external tools need to be used, they can be added by defining a EXTERNAL_TOOLS variable 
# Bootstrap the build by downloading additional tools
.PHONY: bootstrap
bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "Installing $$tool" ; \
		go get -u $$tool; \
	done

.PHONY: install-dep
install-dep:
	@echo "--> Installing external dependencies for building node-disk-manager"
	@sudo $(PWD)/build/install-dep.sh

.PHONY: install-test-infra
install-test-infra:
	@echo "--> Installing test infra for running integration tests"
	# installing test infrastructure is dependent on the platform
	$(PWD)/build/install-test-infra.sh ${XC_ARCH}

.PHONY: header
header:
	@echo "----------------------------"
	@echo "--> node-disk-manager       "
	@echo "----------------------------"
	@echo

# -composite: avoid "literal copies lock value from fakePtr"
.PHONY: vet
vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet -composites

.PHONY: fmt
fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l


.PHONY: controller-gen
controller-gen:
	TMP_DIR=$(shell mktemp -d) && cd $$TMP_DIR && go mod init tmp && go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 && rm -rf $$TMP_DIR;

manifests: controller-gen
	@echo "+ Generating NDM manifest"
	$(PWD)/build/generate-manifests.sh

# shellcheck target for checking shell scripts linting
.PHONY: shellcheck
shellcheck: getshellcheck
	find . -type f -name "*.sh" | grep -v "./vendor/*" | xargs /tmp/shellcheck-stable/shellcheck

.PHONY: getshellcheck
getshellcheck:
	wget -c 'https://github.com/koalaman/shellcheck/releases/download/stable/shellcheck-stable.linux.x86_64.tar.xz' --no-check-certificate -O - | tar -xvJ -C /tmp/
.PHONY: version
version:
	@echo $(VERSION)

.PHONY: test
test: 	vet fmt
	@echo "--> Running go test";
	$(PWD)/build/test.sh ${XC_ARCH}

.PHONY: integration-test
integration-test:
	@echo "--> Running integration test"
	$(PWD)/build/integration-test.sh ${XC_ARCH}

.PHONY: Dockerfile.ndm
Dockerfile.ndm: ./build/ndm-daemonset/Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

.PHONY: Dockerfile.ndo
Dockerfile.ndo: ./build/ndm-operator/Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

.PHONY: Dockerfile.exporter
Dockerfile.exporter: ./build/ndm-exporter/Dockerfile.in
	sed -e 's|@BASEIMAGE@|$(BASEIMAGE)|g' $< >$@

.PHONY: build.ndm
build.ndm:
	@echo '--> Building node-disk-manager binary...'
	@pwd
	@CTLNAME=${NODE_DISK_MANAGER} BUILDPATH=${BUILD_PATH_NDM} sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.ndm
docker.ndm: build.ndm Dockerfile.ndm
	@echo "--> Building docker image for ndm-daemonset..."
	@sudo docker build -t "$(DOCKER_IMAGE_NDM)" ${DBUILD_ARGS} -f Dockerfile.ndm .
	@echo "--> Build docker image: $(DOCKER_IMAGE_NDM)"
	@echo

.PHONY: build.ndo
build.ndo:
	@echo '--> Building node-disk-operator binary...'
	@pwd
	@CTLNAME=${NODE_DISK_OPERATOR} BUILDPATH=${BUILD_PATH_NDO} sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.ndo
docker.ndo: build.ndo Dockerfile.ndo
	@echo "--> Building docker image for ndm-operator..."
	@sudo docker build -t "$(DOCKER_IMAGE_NDO)" ${DBUILD_ARGS} -f Dockerfile.ndo .
	@echo "--> Build docker image: $(DOCKER_IMAGE_NDO)"
	@echo

.PHONY: build.exporter
build.exporter:
	@echo '--> Building node-disk-exporter binary...'
	@pwd
	@CTLNAME=${NODE_DISK_EXPORTER} BUILDPATH=${BUILD_PATH_EXPORTER} sh -c "'$(PWD)/build/build.sh'"
	@echo '--> Built binary.'
	@echo

.PHONY: docker.exporter
docker.exporter: build.exporter Dockerfile.exporter
	@echo "--> Building docker image for ndm-exporter..."
	@sudo docker build -t "$(DOCKER_IMAGE_EXPORTER)" ${DBUILD_ARGS} -f Dockerfile.exporter .
	@echo "--> Build docker image: $(DOCKER_IMAGE_EXPORTER)"
	@echo

# Minimum version of protoc should be 3.12
.PHONY: protos
protos:
	protoc -I . ndm.proto --go_out=plugins=grpc:.

.PHONY: deps
deps: header
	@echo '--> Resolving dependencies...'
	go mod tidy
	go mod verify
	go mod vendor
	@echo '--> Depedencies resolved.'
	@echo

.PHONY: clean
clean: header
	@echo '--> Cleaning directory...'
	rm -rf bin
	rm -rf ${GOPATH}/bin/${NODE_DISK_MANAGER}
	rm -rf ${GOPATH}/bin/${NODE_DISK_OPERATOR}
	rm -rf ${GOPATH}/bin/${NODE_DISK_EXPORTER}
	rm -rf Dockerfile.ndm
	rm -rf Dockerfile.ndo
	rm -rf Dockerfile.exporter
	@echo '--> Done cleaning.'
	@echo

.PHONY: license-check
license-check:
	@echo "--> Checking license header..."
	@licRes=$$(for file in $$(find . -type f -regex '.*\.sh\|.*\.go\|.*Docker.*\|.*\Makefile*' ! -path './vendor/*' ) ; do \
               awk 'NR<=5' $$file | grep -Eq "(Copyright|generated|GENERATED)" || echo $$file; \
       done); \
       if [ -n "$${licRes}" ]; then \
               echo "license header checking failed:"; echo "$${licRes}"; \
               exit 1; \
       fi
	@echo "--> Done checking license."
	@echo

.PHONY: push
push:
	DIMAGE=${IMAGE_ORG}/node-disk-manager-${XC_ARCH} ./build/push;
	DIMAGE=${IMAGE_ORG}/node-disk-operator-${XC_ARCH} ./build/push;
	DIMAGE=${IMAGE_ORG}/node-disk-exporter-${XC_ARCH} ./build/push;


#-----------------------------------------------------------------------------
# Target: docker.buildx.ndm docker.buildx.ndo docker.buildx.exporter
#-----------------------------------------------------------------------------

include Makefile.buildx.mk
=======
# VERSION defines the project version for the bundle. 
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# CHANNELS define the bundle channels used in the bundle. 
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle. 
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMG defines the image:tag used for the bundle. 
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= controller-bundle:$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle ## Generate bundle manifests and metadata, then validate generated files.
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build ## Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
>>>>>>> 841b3476... Initial project structure
