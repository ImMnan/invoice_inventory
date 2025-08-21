
# User-parameterized variables (can be overridden: make build OS=linux ARCH=amd64 INSTALL_PATH=/usr/local/bin)
OS ?= $(shell uname | tr '[:upper:]' '[:lower:]')
ARCH ?= arm64
INSTALL_PATH ?= /usr/local/bin
APP_NAME = lvs

ifeq ($(OS),windows)
    EXTN := .exe
else
    EXTN :=
endif

.PHONY: all build install clean info

all: build

build:
	GOOS=$(OS) GOARCH=$(ARCH) go build -o $(APP_NAME)$(EXTN) main.go

install: build
ifeq ($(OS),windows)
	@echo "Copying binary to $(INSTALL_PATH) and setting PATH for Windows."
	copy $(APP_NAME)$(EXT) $(INSTALL_PATH)\\$(APP_NAME)$(EXTN)
	@echo "Add $(INSTALL_PATH) to your PATH if not already present."
else
	@echo "Copying binary to $(INSTALL_PATH) and making it executable."
	install -m 755 $(APP_NAME)$(EXTN) $(INSTALL_PATH)/$(APP_NAME)$(EXTN)
endif

clean:
	rm -f $(APP_NAME)$(EXTN) $(APP_NAME).exe

info:
	@echo "Building the Project for OS=$(OS), ARCH=$(ARCH), INSTALL_PATH=$(INSTALL_PATH)"
