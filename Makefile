install:
	go get

server:
	cd bin/basic/ && go run main.go

server-auth:
	cd bin/auth/ && go run main.go

test:
	go test

.PHONY: server test
