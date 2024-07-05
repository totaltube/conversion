
totaltube-conversion: $(shell find src/ -type f)
	cd src && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --ldflags='-s -w' -o ../totaltube-conversion

build-docker:
	docker build -t sersh/totaltube-conversion --no-cache .

deploy: build-docker
	docker push sersh/totaltube-conversion

upgrade-sersh: deploy
	ssh ax1 'cd static && docker compose pull conversion && docker compose up -d conversion'
.DEFAULT_GOAL := totaltube-conversion

run-test:
	docker run --rm -it --gpus all -e TOTALTUBE_CONVERSION_API_KEY=1aaPAyzAfcjn6dC4dSmbk0dwT9BdQ2 \
	-e NVIDIA_VISIBLE_DEVICES=all \
	-e NVIDIA_DRIVER_CAPABILITIES=compute,utility,video \
	-e ISCUDA=1 \
	-e TOTALTUBE_CONVERSION_PATH=/data -v $(shell pwd)/test:/test --entrypoint /bin/bash sersh/totaltube-conversion