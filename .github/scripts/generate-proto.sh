#!/bin/sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

export MODULE_NAME=$(go list -m)
export PROTO_DIR="contract/proto"
export GEN_DIR="$PROTO_DIR/generated"

mkdir -p "$GEN_DIR"

for proto in "$PROTO_DIR"/*.proto; do
    protoc \
        --go_out="$GEN_DIR" \
        --go_opt=paths=source_relative,M"$proto"="$MODULE_NAME/$GEN_DIR" \
        --go-grpc_out="$GEN_DIR" \
        --go-grpc_opt=paths=source_relative,M"$proto"="$MODULE_NAME/$GEN_DIR" \
        "$proto"
done

mv "$GEN_DIR"/$PROTO_DIR/*.pb.go "$GEN_DIR"/
rm -rf "$GEN_DIR"/contract