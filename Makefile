PROTOC        := protoc
PROTOC_GEN_GO := $(shell which protoc-gen-go)
PROTOC_GEN_GRPC_GO := $(shell which protoc-gen-go-grpc)

.PHONY: all proto proto-price proto-event clean check-tools

all: proto

proto: proto-price proto-event

proto-price:
	@mkdir -p pb
	@echo "Generating price.proto → pb/"
	$(PROTOC) \
		--proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		proto/price.proto

proto-event:
	@mkdir -p pb
	@echo "Generating event.proto → pb/"
	$(PROTOC) \
		--proto_path=proto \
		--go_out=pb --go_opt=paths=source_relative \
		proto/event.proto

clean:
	@echo "Cleaning generated .pb.go files..."
	@find pb -name "*.pb.go" -delete

check-tools:
	@which protoc-gen-go >/dev/null || echo "⚠️  protoc-gen-go not found"
	@which protoc-gen-go-grpc >/dev/null || echo "⚠️  protoc-gen-go-grpc not found"
