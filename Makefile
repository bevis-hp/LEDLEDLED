# LEDLEDLED build helpers.
#
#   make            - build frontend + native binary into ./bin
#   make run        - run locally against ./dev-config.json
#   make dev        - run the Vite dev server (hot reload) proxying to `make run`
#   make test       - run Go tests
#   make pi         - cross-compile for Raspberry Pi 4 (linux/arm64)
#   make install-pi - first-time install on $(PI_HOST): binary + systemd service
#   make network-pi - copy the two-AP network script to the Pi (run it there)
#   make deploy     - quick update: rebuild, copy binary, restart the service

BINARY   := ledledled
PI_HOST  ?= pi@ledpi.local
PI_DIR   ?= /opt/ledledled

.PHONY: all frontend build run dev test pi install-pi network-pi deploy clean

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

install-pi: pi
	ssh $(PI_HOST) 'mkdir -p ~/ledledled-install'
	scp bin/$(BINARY)-linux-arm64 $(PI_HOST):ledledled-install/$(BINARY)
	scp deploy/ledledled.service deploy/install.sh $(PI_HOST):ledledled-install/
	ssh -t $(PI_HOST) 'bash ~/ledledled-install/install.sh'

network-pi:
	scp deploy/setup-network.sh $(PI_HOST):
	@echo "Now on the Pi (ideally over Ethernet) run:  sudo ./setup-network.sh"

deploy: pi
	scp bin/$(BINARY)-linux-arm64 $(PI_HOST):/tmp/$(BINARY)
	ssh $(PI_HOST) 'sudo install -m 755 /tmp/$(BINARY) $(PI_DIR)/$(BINARY) && sudo systemctl restart $(BINARY)'

clean:
	rm -rf bin
	find internal/webui/dist -mindepth 1 ! -name .gitkeep -delete
