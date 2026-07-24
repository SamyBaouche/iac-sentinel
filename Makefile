run:
	go run .

build:
	go build -o bin/iac-sentinel.exe .

fmt:
	go fmt ./...

test:
	go test ./...

clean:
		powershell -Command "Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue"