.PHONY: build
build: argosay
	docker build . -t argoproj/argosay:v2
	docker push argoproj/argosay:v2

argosay:
	GOOS=linux CGO_ENABLED=0 go build -ldflags '-w -s' -o argosay ./main
