# Go parameters
APP_NAME=lvs
BIN_DIR=/usr/local/bin/
BIN_PATH=$(BIN_DIR)/$(APP_NAME)

.PHONY: all build clean info

all: build

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_PATH) main.go
	chmod 755 $(BIN_PATH)
clean:
	rm -f $(BIN_PATH)
info:
	echo "Building the Project..."

