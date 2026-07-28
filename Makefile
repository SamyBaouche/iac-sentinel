.PHONY: run build fmt test vet clean

run:
	go run ./cmd/iac-sentinel

build:
	go build -ldflags="-X main.Version=0.1.0" -o bin/iac-sentinel ./cmd/iac-sentinel

fmt:
	go fmt ./...

test:
	go test ./... -cover

vet:
	go vet ./...

clean:
	rm -rf bin
