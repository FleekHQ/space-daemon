## Textile Bucket Issues

This example is to highlight some issues that are currently being experience
using textiles threads/buckets.

### Issue 1: Buckets directory returns `false` for `IsDir`.

To replicate this issue, run the `is_dir_issue/main.go` program.
1. Ensure you are currently in this directory.
2. Run `docker-compose up -d`
3. Run `go run is_dir_issue/main.go`. You should get an error with message like `parentFolder's ListPathItem.IsDir should be 'true', but got false` 

Don't forget to run stop docker afterwards with `docker-compose down`

### Issue 2: Textile daemon does not properly shutdown its badger resource.

To replicate this issue, run the `stop_start_issue/main.go` program.
1. Ensure you are currently in this directory.
2. Run `docker-compose up -d`
3. Run `go run stop_start_issue/main.go`. You should get an error similar to this stacktrace
```
Failed to start textile a second time: resource temporarily unavailable
Cannot acquire directory lock on ".buckd/repo/eventstore".  Another process is using this Badger database.
github.com/dgraph-io/badger.acquireDirectoryLock
	/Users/perfect/go/pkg/mod/github.com/dgraph-io/badger@v1.6.1/dir_unix.go:66
github.com/dgraph-io/badger.Open
	/Users/perfect/go/pkg/mod/github.com/dgraph-io/badger@v1.6.1/db.go:230
github.com/ipfs/go-ds-badger.NewDatastore
	/Users/perfect/go/pkg/mod/github.com/ipfs/go-ds-badger@v0.2.4/datastore.go:137
github.com/textileio/go-threads/db.newDefaultDatastore
	/Users/perfect/go/pkg/mod/github.com/textileio/go-threads@v0.1.21/db/options.go:32
github.com/textileio/go-threads/db.NewManager
	/Users/perfect/go/pkg/mod/github.com/textileio/go-threads@v0.1.21/db/manager.go:46
github.com/textileio/go-threads/api.NewService
	/Users/perfect/go/pkg/mod/github.com/textileio/go-threads@v0.1.21/api/service.go:52
github.com/textileio/textile/core.NewTextile
	/Users/perfect/go/pkg/mod/github.com/textileio/textile@v1.0.13-0.20200707162859-d8131f5afba4/core/core.go:197
main.StartTextile
	/Users/perfect/Terminal/space-poc/examples/bucketIssues/stop_start_issue/main.go:15
main.main
	/Users/perfect/Terminal/space-poc/examples/bucketIssues/stop_start_issue/main.go:48
runtime.main
	/Users/perfect/go/go1.14.3/src/runtime/proc.go:203
runtime.goexit
	/Users/perfect/go/go1.14.3/src/runtime/asm_amd64.s:1373
exit status 1

```