version := 1.0
SOURCES := $(shell find src -type f)

# Собираем бинарь через docker-контейнер sersh/golang-builder
bin/totaltube-conversion: $(SOURCES)
	@mkdir -p bin
	@docker run \
		-v /home/sersh/projects/fasttube/totaltube-conversion:/src \
		-v /home/sersh/go/docker/bin:/go/bin \
		-v /home/sersh/go/docker/pkg:/go/pkg \
		-v /home/sersh/go/docker/src:/go/src \
		-v /home/sersh/go/docker/.cache:/root/.cache \
		--name ttt-minion-build -t --rm sersh/golang-builder \
		make -C /src bin/docker-totaltube-conversion
	@cp bin/docker-totaltube-conversion bin/totaltube-conversion && rm -f bin/docker-totaltube-conversion

# ВНУТРИ докера: реальная сборка статического бинаря
bin/docker-totaltube-conversion: $(SOURCES)
	cd src && CGO_ENABLED=1 CGO_LDFLAGS="-static" GOOS=linux GOARCH=amd64 go build \
		--tags="netgo osusergo" \
		--ldflags='-s -w -X main.version=$(version) -extldflags=-static' \
		-o ../bin/docker-totaltube-conversion .

build-docker: bin/totaltube-conversion
	docker build -t sersh/totaltube-conversion . \
	&& touch build-docker

deploy: build-docker
	docker push sersh/totaltube-conversion

upgrade-sersh: deploy
	ssh ax1 'cd static && docker compose pull conversion && docker compose up -d conversion'
.DEFAULT_GOAL := bin/totaltube-conversion

run-test: build-docker
	docker run --rm -it --gpus all --network=host -e TOTALTUBE_CONVERSION_API_KEY=1aaPAyzAfcjn6dC4dSmbk0dwT9BdQ2 \
	-e NVIDIA_VISIBLE_DEVICES=all \
	-e NVIDIA_DRIVER_CAPABILITIES=compute,utility,video \
	-e ISCUDA=1 \
	-e TOTALTUBE_CONVERSION_PATH=/data -v $(shell pwd)/test:/test --entrypoint /bin/bash sersh/totaltube-conversion

run-test-server: build-docker
	docker run --rm -it --gpus all --network=host \
	-e TOTALTUBE_CONVERSION_API_KEY=1aaPAyzAfcjn6dC4dSmbk0dwT9BdQ2 \
	-e NVIDIA_VISIBLE_DEVICES=all \
	-e TOTALTUBE_CONVERSION_PORT=9098 \
	-e NVIDIA_DRIVER_CAPABILITIES=compute,utility,video \
	-e ISCUDA=1 \
	-e TOTALTUBE_CONVERSION_PATH=/data -v $(shell pwd)/test:/test sersh/totaltube-conversion