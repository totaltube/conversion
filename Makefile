
totaltube-conversion: $(shell find src/ -type f)
	cd src && GOOS=linux GOARCH=amd64 go build --ldflags='-s -w' -o ../totaltube-conversion

build-docker: totaltube-conversion
	docker build -t sersh/totaltube-conversion . && touch build-docker

deploy: build-docker
	docker push sersh/totaltube-conversion

.DEFAULT_GOAL := totaltube-conversion