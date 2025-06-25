@echo off
echo Generating protobuf files...

protoc -I proto ^
    --go_out=genproto ^
    --go_opt=paths=source_relative ^
    --go-grpc_out=genproto ^
    --go-grpc_opt=paths=source_relative ^
    proto/rating.proto

echo Proto files generated successfully!