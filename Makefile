.PHONY: run build fmt test vet clean

run:
	go run .

build:
	go build -o bin/iac-sentinel .

fmt:
	go fmt ./...

test:
	go test ./... -cover

vet:
	go vet ./...

clean:
	rm -rf bin
