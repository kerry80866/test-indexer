PROTO_DIR=api/price
PROTO_FILE=price.proto

proto:
	protoc \
		--proto_path=. \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		$(PROTO_DIR)/$(PROTO_FILE)
