PROG_NAME := "schema-registry"
GIT_VERSION := $(shell git log -1 --pretty=format:"%h (%ci)" .)

setup: install tls compiledaemon

install:
	@if command -v glide &> /dev/null; then \
		echo >&2 'Installing library dependences'; \
		glide install; \
	else \
		echo >&2 'Glide required: https://glide.sh'; \
		exit 1; \
	fi

test-install: install
	go get golang.org/x/tools/cmd/cover
	go get github.com/mattn/goveralls

build-install: install test-install
	go get github.com/mitchellh/gox

tls:
	@if [ ! -a cert.pem ]; then \
		echo >&2 'Creating self-signed TLS certs.'; \
		go run $(shell go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost; \
	fi

compiledaemon:
	@if command -v CompileDaemon &> /dev/null; then \
		echo >&2 'Getting CompileDaemon for auto-reload.'; \
		go get github.com/githubnemo/CompileDaemon; \
	fi

watch:
	CompileDaemon \
		-build="make build" \
		-command="$(PROG_NAME)" \
		-graceful-kill=true \
		-exclude-dir=.git \
		-exclude-dir=vendor \
		-color=true

test:
	go test -cover $(glide novendor)

bench:
	go test -run=none -bench=. -benchmem ./...

build:
	go build \
		-ldflags "-X \"main.buildVersion=$(GIT_VERSION)\"" \
		-o $(GOPATH)/bin/$(PROG_NAME) \
		./cmd/schema-registry

dist-build:
	mkdir -p dist

	gox -output="./dist/{{.OS}}-{{.Arch}}/$(PROG_NAME)" \
		-ldflags "-X \"main.buildVersion=$(GIT_VERSION)\"" \
		-os "windows linux darwin" \
		-arch "amd64" \
		./cmd/schema-registry > /dev/null

dist-zip:
	cd dist && zip $(PROG_NAME)-darwin-amd64.zip darwin-amd64/*
	cd dist && zip $(PROG_NAME)-linux-amd64.zip linux-amd64/*
	cd dist && zip $(PROG_NAME)-windows-amd64.zip windows-amd64/*

dist: dist-build dist-zip

.PHONY: test assets build dist
