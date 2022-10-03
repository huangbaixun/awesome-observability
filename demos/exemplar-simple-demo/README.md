
# Build

GOOS=linux GOARCH=amd64 go build -o http-server server/*.go
docker rmi fatsheep9146/demo-http-server:v0.1
docker build -t fatsheep9146/demo-http-server:v0.1 -f dockerfile.server .

# Ref

1. https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/examples/demo
