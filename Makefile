SERVER ?= root@antech
BIN    := hawkeye

.PHONY: build deploy

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BIN) ./cmd/hawkeye

deploy: build
	rsync -avz $(BIN) $(SERVER):/tmp/$(BIN)
	ssh -t $(SERVER) 'sudo install -m 755 /tmp/$(BIN) /usr/local/bin/$(BIN)'
