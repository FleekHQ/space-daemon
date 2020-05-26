# space-poc


Research repository to test and try things
this code later needs to be rewritten or moved to the actual
repositories that will be used for implementation.


# Running the Space Binary

Binary should run in a folder with a space.json config file with the following settings:
```json
{
  "space": {
    "folderPath": "/path/to/shared/folder",
    "rpcPort": 9999,
    "storePath": "~/.fleek-space"
  }
}
```

If you still have issues, try setting the env var SPACE_APP_DIR
`export SPACE_APP_DIR=/path/to/the/space/binary`

# POC Package Structure
Loosely based on these resources:
https://github.com/golang-standards/project-layout


Note: For POC purposes the package structure is not as important but when we do migrate to a real folder we want to
structure as much as possible following standards  

`/api` Folder structure for REST API.
`/cmd` Entry point directory for all binaries this repo handles. E.g cmd/{binary-name}/main.go
`/config` Global Config code
`/core` Directory for core stuff like watcher service and threads watcher
`/logger` Directory for app logging 
`/examples` Directory playground for general examples and drafts

Other Potential directories to consider adding
`/shared` or  `/common` In case we want to share libs or logic among several binaries like 
ipfs logic and other interactions.

