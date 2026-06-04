GOCACHE ?= $(CURDIR)/.gocache
export GOCACHE
BINARY_NAME := config-api
DIST_DIR := dist
PACKAGE_STAGING := $(DIST_DIR)/package

.PHONY: build test vet check clean \
	build-linux-amd64 build-linux-arm64 \
	package-linux-amd64 package-linux-arm64

build:
	mkdir -p bin
	go build -buildvcs=false -trimpath -o bin/$(BINARY_NAME) ./cmd/config-api

test:
	go test ./...

vet:
	go vet ./...

check: test vet build

build-linux-amd64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/config-api

build-linux-arm64:
	mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/config-api

package-linux-amd64: build-linux-amd64
	mkdir -p $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64
	cp $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/$(BINARY_NAME)
	cp deploy/$(BINARY_NAME).service $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/$(BINARY_NAME).service
	cp packaging/install.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/install.sh
	cp packaging/uninstall.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/uninstall.sh
	cp README.md $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/README.md
	chmod 0755 $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/install.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-amd64/uninstall.sh
	tar -C $(PACKAGE_STAGING) -czf $(DIST_DIR)/$(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64

package-linux-arm64: build-linux-arm64
	mkdir -p $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64
	cp $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/$(BINARY_NAME)
	cp deploy/$(BINARY_NAME).service $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/$(BINARY_NAME).service
	cp packaging/install.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/install.sh
	cp packaging/uninstall.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/uninstall.sh
	cp README.md $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/README.md
	chmod 0755 $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/install.sh $(PACKAGE_STAGING)/$(BINARY_NAME)-linux-arm64/uninstall.sh
	tar -C $(PACKAGE_STAGING) -czf $(DIST_DIR)/$(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64

clean:
	rm -rf bin $(DIST_DIR) .gocache
