
for /D %%d in ("proto\*") do (
    for %%f in ("%%d\*.proto") do (
        protoc --proto_path="%%d" --go_out="%%d/../../../" "%%f"
    )
)
