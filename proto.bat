
for /D %%d in ("proto\*") do (
    for %%f in ("%%d\*.proto") do (
        echo Processing %%f...
        protoc --proto_path="%%d" --go_out="%%d/../../../" "%%f"
    )
)