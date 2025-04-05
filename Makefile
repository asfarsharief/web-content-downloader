#Makefile to run the application, run test cases, and build docker image 
#=======================================================================

dep:
	@go get ./...

run:
	go run cmd/main.go trigger -p ${PATH} -o ${OUTPUT}

build:
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a --installsuffix cgo -o bin/app  cmd/main.go

test:
	@go test -coverprofile=.code_coverage.out ./...
