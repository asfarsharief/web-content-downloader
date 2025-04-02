#Makefile to run the application, run test cases, and build docker image 
#=======================================================================
APP_NAME = cli-app

dep:
	@go get ./...

build:
	go build -o $(APP_NAME) main.go
	
run: build
	./$(APP_NAME) trigger $(ARGS)