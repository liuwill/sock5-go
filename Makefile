install:
	go get

server:
	cd bin/ && go run main.go

test:
	go test

.PHONY: server test
