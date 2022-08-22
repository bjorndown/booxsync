# Boox sync

Sync a local folder to a Boox Note Air 2 library via its BooxDrop HTTP API

## How

```sh
# clone repo
cd cmd/booxsync
go build -ldflags "-s -w" .
cat << EOF > config.json
{
  "host": "http://192.168.1.40:8085",
  "syncRoot": "/home/foo/whatever",
  "pathsToSkip": [
    "/dont-care"
  ]
}
EOF
./booxsync -dryRun # 
```
