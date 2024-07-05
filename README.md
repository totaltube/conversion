# Totaltube conversion server
This is repository of totaltube conversion server for docker

#### Running on your server
1. Install [docker](https://docs.docker.com/engine/install/)
2. Run it with docker or use [docker-compose.yml](docker-compose.yml) with [docker-compose](https://docs.docker.com/compose/)
```shell
# running on 8081 port with api key some-api-key
docker run -d -e TOTALTUBE_CONVERSION_API_KEY='some-api-key' -p 8081:8080 --name totaltube-conversion docker.io/sersh/totaltube-conversion
```