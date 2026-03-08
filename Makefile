# --- Variable definitions ---
BINARY_NAME=procmon
BUILD_DIR=build
MAIN_PATH=./cmd

# --- Detect OS to handle .exe suffix ---
ifeq ($(OS),Windows_NT)
    BINARY_EXT=.exe
else
    BINARY_EXT=
endif

# Determine host OS/ARCH for naming the local build output
HOST_GOOS := $(shell go env GOOS)
HOST_GOARCH := $(shell go env GOARCH)

# Note: Define TARGET under BUILD_DIR
TARGET=$(BUILD_DIR)/$(BINARY_NAME)_$(HOST_GOOS)_$(HOST_GOARCH)$(BINARY_EXT)

# Define the directory creation command at the top of the Makefile
ifeq ($(OS),Windows_NT)
    # Windows cmd doesn't support -p, and mkdir errors if the directory already exists, so add a guard
    # Alternatively, cmd's mkdir is recursive by default
    MKDIR = if not exist $(BUILD_DIR) mkdir

    # Windows clean commands
    RM = del /F /Q
    RMRF = rmdir /S /Q
    NULLDEV = >nul 2>&1
else
    MKDIR = mkdir -p

    # Unix clean commands
    RM = rm -f
    RMRF = rm -rf
    NULLDEV =
endif

# Cross-compilation environment variable setup:
# - Unix (Linux/macOS) shells support `GOOS=... GOARCH=...` prefix
# - Windows cmd requires `set VAR=...&&` syntax
ifeq ($(OS),Windows_NT)
    GOENV = set "GOOS=$(GOOS)"&& set "GOARCH=$(GOARCH)"&&
else
    GOENV = GOOS=$(GOOS) GOARCH=$(GOARCH)
endif

.PHONY: all build clean run help windows linux darwin release

# Default target is build
all: build

## build: compile and store into build directory
ifeq ($(OS),Windows_NT)
build:
	@echo "Building via build_windows.bat..."
	@build_windows.bat
else
build:
	@echo "Building via build_unix.sh..."
	@chmod +x build_unix.sh 2>/dev/null || true
	@./build_unix.sh
endif

## run: run the binary from the build directory
run: build
	$(TARGET) -p "chrome" -d 10

## clean: remove build artifacts from build dir and root
clean:
	@echo "Cleaning build artifacts..."
	-@$(RM) $(BINARY_NAME) $(BINARY_NAME).exe $(NULLDEV)
	-@$(RMRF) $(BUILD_DIR) $(NULLDEV)
	@echo "Clean complete."

# --- Cross-compilation targets (all output to BUILD_DIR) ---

## windows: cross-compile Windows 64-bit build
windows: GOOS = windows
windows: GOARCH = amd64
windows:
	@$(MKDIR) $(BUILD_DIR)
	@echo "Cross-compiling Windows build..."
ifeq ($(OS),Windows_NT)
	@$(GOENV) build_windows.bat
else
	@chmod +x build_unix.sh 2>/dev/null || true
	@$(GOENV) ./build_unix.sh
endif

## linux: cross-compile Linux 64-bit build
linux: GOOS = linux
linux: GOARCH = amd64
linux:
	@$(MKDIR) $(BUILD_DIR)
	@echo "Cross-compiling Linux build..."
ifeq ($(OS),Windows_NT)
	@$(GOENV) build_windows.bat
else
	@chmod +x build_unix.sh 2>/dev/null || true
	@$(GOENV) ./build_unix.sh
endif

## darwin: cross-compile macOS build
darwin: GOOS = darwin
darwin: GOARCH = amd64
darwin:
	@$(MKDIR) $(BUILD_DIR)
	@echo "Cross-compiling macOS build..."
ifeq ($(OS),Windows_NT)
	@$(GOENV) build_windows.bat
else
	@chmod +x build_unix.sh 2>/dev/null || true
	@$(GOENV) ./build_unix.sh
endif


## release: build all platform binaries
release: clean
	@$(MAKE) windows
	@$(MAKE) linux
	@$(MAKE) darwin
	@echo "All builds ready. See $(BUILD_DIR) directory."

