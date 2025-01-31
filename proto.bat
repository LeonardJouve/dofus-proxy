
for /D %%d in ("proto\*") do (
    for %%f in ("%%d\*.proto") do (
        protoc --proto_path="%PROTOBUF_INCLUDE_PATH%" --proto_path="%%d" --go_out="%%d/../../../" "%%f"
    )
)
