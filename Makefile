
bin:
	go build -ldflags "-s -w" -o bin/booxsync cmd/booxsync/main.go
run:
	go run cmd/booxsync/main.go
clean:
	rm -rf bin
test:
	cd pkg/booxsync && go test
