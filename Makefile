# LEDLEDLED build helpers.
#
#   make            - build frontend + native binary into ./bin
#   make run        - run locally against ./dev-config.json
#   make dev        - run the Vite dev server (hot reload) proxying to `make run`
#   make test       - run Go tests
#   make pi         - cross-compile for Raspberry Pi 4 (linux/arm64)
#   make deploy     - build for Pi, copy to $(PI_HOST) and restart the service

BINARY   := ledledled
PI_HOST  ?= pi@ledpi.local
PI_DIR   ?= /opt/ledledled

.PHONY: all frontend build run dev test pi deploy clean

all: frontend build

frontend:
	cd frontend && npm install && npm run build

build:
	go build -o bin/$(BINARY) ./cmd/ledledled

run:
	go run ./cmd/ledledled -config dev-config.json -listen :8080

dev:
	cd frontend && npm run dev

test:
	go test ./...

pi: frontend
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
		-o bin/$(BINARY)-linux-arm64 ./cmd/ledledled

deploy: pi
	scp bin/$(BINARY)-linux-arm64 $(PI_HOST):/tmp/$(BINARY)
	ssh $(PI_HOST) 'sudo install -m 755 /tmp/$(BINARY) $(PI_DIR)/$(BINARY) && sudo systemctl restart $(BINARY)'

clean:
	rm -rf bin
	find internal/webui/dist -mindepth 1 ! -name .gitkeep -delete
