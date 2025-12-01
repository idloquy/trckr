VERSION = $(shell git describe --tags)
OUT_DIR = out

GO_LDFLAGS = -s -w -X 'main.Version=$(VERSION)'
GO_FLAGS = -trimpath -ldflags="$(GO_LDFLAGS)"

export CGO_ENABLED=1

default: host

host:
	go build $(GO_FLAGS) -o "$(OUT_DIR)/" ./...

linux_amd64:
	CC="zig cc -target x86_64-linux-gnu" GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o "$(OUT_DIR)/$@/" ./...

linux_arm64:
	CC="zig cc -target aarch64-linux-gnu" GOOS=linux GOARCH=arm64 go build $(GO_FLAGS) -o "$(OUT_DIR)/$@/" ./...

linux_armv7:
	CC="zig cc -target arm-linux-gnueabihf" GOOS=linux GOARCH=arm GOARM=7 go build $(GO_FLAGS) -o "$(OUT_DIR)/$@/" ./...

windows_amd64:
	CC="zig cc -target x86_64-windows-gnu" GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o "$(OUT_DIR)/$@/" ./...

windows_arm64:
	CC="zig cc -target aarch64-windows-gnu" GOOS=windows GOARCH=arm64 go build $(GO_FLAGS) -o "$(OUT_DIR)/$@/" ./...
