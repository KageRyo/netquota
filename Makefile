.PHONY: fmt vet test build icons check

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/netquota ./cmd/netquota

icons:
	go run ./tools/iconassets

check: vet test build
