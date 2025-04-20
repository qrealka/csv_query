# Docker image for multi‐arch Go 1.24.1
GO_IMAGE := golang:1.24.1

# Output directory for built binaries
BUILD_DIR := buildDir
CMD_PATH   := ./cmd/brightbeam
BIN_NAME   := brightbeam

.PHONY: all tests build clean

all: tests build

tests:
	@docker run --rm \
      -v "$(PWD)":/src \
      -w /src \
      $(GO_IMAGE) \
      go test ./... && go test -race ./...
      

build: $(BUILD_DIR)
	docker run --rm \
      -v "$(PWD)":/src \
      -w /src \
	  -e CGO_ENABLED=0 \
      $(GO_IMAGE) \
      go build -buildvcs=false -tags osusergo,netgo \
	   -trimpath -ldflags="-extldflags=-static" \
	   -o $(BUILD_DIR)/$(BIN_NAME) $(CMD_PATH)

# ensure build directory exists
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# clean build artifacts
clean:
	rm -rf $(BUILD_DIR)