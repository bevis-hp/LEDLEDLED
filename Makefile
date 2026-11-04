# LEDLEDLED build helpers.
#
#   make            - build frontend + native binary into ./bin
#   make run        - run locally against ./dev-config.json
#   make dev        - run the Vite dev server (hot reload) proxying to `make run`
#   make test       - run Go tests
#   make pi         - cross-compile for Raspberry Pi 4 (linux/arm64)
#   make install    - on the Pi: build, install the systemd service and
#                     (re)start it. Also how you update after `git pull`.

BINARY   := ledledled
VERSION  := $(shell git describe --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -X main.version=$(VERSION)

.PHONY: all frontend build run dev test pi install clean

all: frontend build

frontend:
	cd frontend && npm install && npm run build

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/ledledled

run:
	go run ./cmd/ledledled -config dev-config.json -listen :8080

dev:
	cd frontend && npm run dev

test:
	go test ./...

pi: frontend
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w $(LDFLAGS)" \
		-o bin/$(BINARY)-linux-arm64 ./cmd/ledledled

# Run on the Pi itself. npm ci keeps package-lock.json untouched so
# `git pull` stays clean.
install:
	cd frontend && npm ci && npm run build
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w $(LDFLAGS)" -o bin/$(BINARY) ./cmd/ledledled
	sudo deploy/install.sh bin/$(BINARY)

clean:
	rm -rf bin
	find internal/webui/dist -mindepth 1 ! -name .gitkeep -delete
