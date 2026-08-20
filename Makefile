.PHONY: fmt vet test build check

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/netquota ./cmd/netquota

check: vet test build

