.PHONY: run build fmt test vet clean

run:
	go run ./cmd/tfguard

build:
	go build -ldflags="-X main.Version=0.1.0" -o bin/tfguard ./cmd/tfguard

fmt:
	go fmt ./...

test:
	go test ./... -cover

vet:
	go vet ./...

clean:
	rm -rf bin
