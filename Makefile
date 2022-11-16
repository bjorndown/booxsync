bin: linux-bin darwin-bin windows-bin

linux-bin:
	GOOS=linux go build -ldflags "-s -w" -o bin/booxsync-linux-amd64 cmd/booxsync/main.go
darwin-bin:
	GOOS=darwin go build -ldflags "-s -w" -o bin/booxsync-darwin-amd64 cmd/booxsync/main.go
windows-bin:
	GOOS=windows go build -ldflags "-s -w" -o bin/booxsync-windows-amd64.exe cmd/booxsync/main.go
run:
	go run cmd/booxsync/main.go
clean:
	rm -rf bin
test:
	cd pkg/booxsync && go test
