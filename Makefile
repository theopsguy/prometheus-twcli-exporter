.DEFAULT_GOAL := build

.PHONY:fmt vet build
fmt:
	go fmt ./...
vet: fmt
	go vet ./...
build: vet
	go build
clean:
	go clean
test:
	go test -json -v ./... | go tool gotestfmt
