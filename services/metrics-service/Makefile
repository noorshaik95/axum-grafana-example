.PHONY: proto build test clean

# Generate protobuf Go code from /proto/metrics.proto into api/proto/.
proto:
	@echo "Generating protobuf code..."
	@mkdir -p api/proto
	PATH=$$PATH:$(shell go env GOPATH)/bin protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		-I ../../proto \
		../../proto/metrics.proto
	@mv -f metrics.pb.go metrics_grpc.pb.go api/proto/ 2>/dev/null || true

build: proto
	go build ./...

test:
	go test -count=1 ./...

clean:
	rm -rf api/proto/*.pb.go
