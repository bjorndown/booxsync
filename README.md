# Boox sync

Sync a local folder to a Boox Note Air 2 library via its BooxDrop HTTP API

## How

```sh
# clone repo, cd into it
yarn
cat << EOF > config.json 
{
  "host": "http://192.168.1.40:8085",
  "syncRoot": "/home/foo/whatever",
  "skipPaths": [
    "/dont-care/"
  ]
}
EOF
yarn sync
```
