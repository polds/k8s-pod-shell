APP ?= kubeshell-web
VERSION ?= dev
GIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

.PHONY: deps
deps:
	go mod tidy
	cd web && npm install

.PHONY: build
build:
	cd web && npm run build
	mkdir -p cmd/kubeshell-web/web/dist
	cp -R web/dist/* cmd/kubeshell-web/web/dist/
	GOFLAGS="-trimpath" go build -ldflags "-X main.version=$(VERSION) -X main.gitSHA=$(GIT_SHA)" -o bin/$(APP) ./cmd/kubeshell-web

.PHONY: test
test:
	go test ./...
	cd web && npm run test
