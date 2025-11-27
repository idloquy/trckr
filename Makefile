VERSION = $(shell git describe --tags)
OUT_DIR = out

GO_LDFLAGS = -s -w -X "main.Version=$(VERSION)"
GO_FLAGS = -trimpath -ldflags="$(GO_LDFLAGS)"

default: $(OUT_DIR)
	go build $(GO_FLAGS) -o "$(OUT_DIR)" ./...

$(OUT_DIR):
	mkdir -p "$(OUT_DIR)"
