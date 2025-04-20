# Docker image for multi‐arch Go 1.24.1
GO_IMAGE := golang:1.24.1
CURL_IMAGE := alpine/curl:latest

# Output directory for built binaries and data files
BUILD_DIR := buildDir
CMD_PATH   := ./cmd/brightbeam
BIN_NAME   := brightbeam

JSON_URL := https://hiring.brightbeam.engineering/dublin-trees.json
CSV_URL  := https://hiring.brightbeam.engineering/dublin-property.csv
JSON_FILE := $(BUILD_DIR)/dublin-trees.json
CSV_FILE  := $(BUILD_DIR)/dublin-property.csv


.PHONY: all tests build clean download-data run

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
	@echo "Build complete. Binary available at $(BUILD_DIR)/$(BIN_NAME)."

# ensure build directory exists
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# download input data files
download-data: $(BUILD_DIR)
	@echo "Downloading data files to $(BUILD_DIR)..."
	@docker run --rm \
	  -v "$(PWD)/$(BUILD_DIR)":/buildDir \
	  $(CURL_IMAGE) \
	  -Lo /buildDir/$(notdir $(JSON_FILE)) $(JSON_URL)
	@docker run --rm \
	  -v "$(PWD)/$(BUILD_DIR)":/buildDir \
	  $(CURL_IMAGE) \
	  -Lo /buildDir/$(notdir $(CSV_FILE)) $(CSV_URL)

# build and download data (ready to run)
run: build download-data
	@echo "Build complete and data downloaded to $(BUILD_DIR)."
	@cd $(BUILD_DIR) && ./$(BIN_NAME)

# clean build artifacts and downloaded data
clean:
	rm -rf $(BUILD_DIR)