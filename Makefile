# Hive 2.0 build system
.PHONY: build build-agent build-server test clean

GO := go
GOFLAGS := -ldflags="-s -w"
BIN_DIR := bin
AGENT_BIN := $(BIN_DIR)/hive-agent
SERVER_BIN := $(BIN_DIR)/hive-server

build: build-agent build-server

build-agent:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(AGENT_BIN) ./cmd/hive
	@echo "agent -> $(AGENT_BIN)"

build-server:
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(SERVER_BIN) ./cmd/hive
	@echo "server -> $(SERVER_BIN)"

# cross-compile for Windows agent
build-agent-win:
	@mkdir -p $(BIN_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BIN_DIR)/hive-agent.exe ./cmd/hive
	@echo "windows agent -> $(BIN_DIR)/hive-agent.exe"

test:
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
