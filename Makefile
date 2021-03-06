
GIT_HOST = sigs.k8s.io
PWD := $(shell pwd)
BASE_DIR := $(shell basename $(PWD))
TESTARGS_DEFAULT := "-v"
export TESTARGS ?= $(TESTARGS_DEFAULT)

HAS_LINT := $(shell command -v golint;)
HAS_GOX := $(shell command -v gox;)
HAS_YQ := $(shell command -v yq;)
GOX_PARALLEL ?= 3
TARGETS ?= darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 linux/ppc64le
DIST_DIRS         = find * -type d -exec

GENERATE_YAML_PATH=cmd/clusterctl/examples/openstack
GENERATE_YAML_EXEC=generate-yaml.sh
GENERATE_YAML_TEST_FOLDER=dummy-make-auto-test

GOOS ?= $(shell go env GOOS)
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)
GOFLAGS   :=
TAGS      :=
LDFLAGS   := "-w -s -X 'main.version=${VERSION}'"
REGISTRY ?= k8scloudprovider

.PHONY: vendor
vendor: ## Runs go mod to ensure proper vendoring.
	./hack/update-vendor.sh

build: binary images

binary: manager clusterctl

manager:
	CGO_ENABLED=0 GOOS=$(GOOS) go build -v \
		-ldflags $(LDFLAGS) \
		-o bin/manager \
		cmd/manager/main.go

clusterctl:
	CGO_ENABLED=0 GOOS=$(GOOS) go build \
		-ldflags $(LDFLAGS) \
		-o bin/clusterctl \
		cmd/clusterctl/main.go

test: unit functional generate_yaml_test

check: vendor fmt vet lint

generate_yaml_test:
ifndef HAS_YQ
	go get github.com/mikefarah/yq
	echo "installing yq"
endif
	# Create a dummy file for test only
	echo 'clouds' > dummy-clouds-test.yaml
	$(GENERATE_YAML_PATH)/$(GENERATE_YAML_EXEC) -f dummy-clouds-test.yaml openstack ubuntu $(GENERATE_YAML_TEST_FOLDER)
	# the folder will be generated under same folder of $(GENERATE_YAML_PATH)
	rm -fr $(GENERATE_YAML_PATH)/$(GENERATE_YAML_TEST_FOLDER)
	rm dummy-clouds-test.yaml

unit: generate vendor
	go test -tags=unit ./pkg/... ./cmd/... $(TESTARGS)

functional:
	@echo "$@ not yet implemented"

fmt:
	hack/verify-gofmt.sh

lint:
ifndef HAS_LINT
		go get -u golang.org/x/lint/golint
		echo "installing golint"
endif
	hack/verify-golint.sh

vet:
	go vet ./pkg/... ./cmd/...

cover: generate vendor
	go test -tags=unit ./pkg/... ./cmd/... -cover

docs:
	@echo "$@ not yet implemented"

godoc:
	@echo "$@ not yet implemented"

releasenotes:
	@echo "Reno not yet implemented for this repo"

translation:
	@echo "$@ not yet implemented"

# Do the work here

# Set up the development environment
env:
	@echo "PWD: $(PWD)"
	@echo "BASE_DIR: $(BASE_DIR)"
	go version
	go env

clean:
	rm -rf _dist bin/manager bin/clusterctl

realclean: clean
	rm -rf vendor
	if [ "$(GOPATH)" = "$(GOPATH_DEFAULT)" ]; then \
		rm -rf $(GOPATH); \
	fi

shell:
	$(SHELL) -i

# Generate code
generate: manifests
	go generate ./pkg/... ./cmd/...

manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go crd

images: openstack-cluster-api-controller clusterctl-image

openstack-cluster-api-controller: generate manifests
	docker build . -f cmd/manager/Dockerfile --network=host -t "$(REGISTRY)/openstack-cluster-api-controller:$(VERSION)"

clusterctl-image: generate manifests
	docker build . -f cmd/clusterctl/Dockerfile --network=host -t "$(REGISTRY)/openstack-cluster-api-clusterctl:$(VERSION)"

upload-images: images
	@echo "push images to $(REGISTRY)"
	docker login -u="$(DOCKER_USERNAME)" -p="$(DOCKER_PASSWORD)";
	docker push $(REGISTRY)/openstack-cluster-api-controller:$(VERSION)
	docker push $(REGISTRY)/openstack-cluster-api-clusterctl:$(VERSION)

version:
	@echo ${VERSION}

.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross: vendor
ifndef HAS_GOX
	go get -u github.com/mitchellh/gox
endif
	CGO_ENABLED=0 gox -parallel=$(GOX_PARALLEL) -output="_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)' $(GIT_HOST)/$(BASE_DIR)/cmd/openstack-machine-controller/

.PHONY: dist
dist: build-cross
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf cluster-api-provider-openstack-$(VERSION)-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r cluster-api-provider-openstack-$(VERSION)-{}.zip {} \; \
	)

.PHONY: build clean cover vendor docs fmt functional lint realclean \
	relnotes test translation version build-cross dist manifests
