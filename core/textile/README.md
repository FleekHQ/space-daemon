# Textile Wrappers

This package contains wrappers around Textile Threads and Buckets.

## Usage

### Initialization and Startup

Initialize Space's Textile Client by runing

```go
client := textile.NewClient(store)
client.Start(ctx, config)
```

This will start the Textile connection, try to authenticate to the Hub, create a thread for metadata and a default bucket if there's none.

#### TODO

- Separate the initialization logic into a new function that can be called from an imported "wallet". That way the initialized metadata and initial buckets can be pulled from the Hub if they exist.

## Internal State

Textile Client holds the following objects:

- store: A connection to the store. Used to fetch keys and store the meta thread threadID.
- threads: A connection to Textile's Thread Client, initiated after startup.
- bucketsClient: A connection to Textile's Bucket Client, initiated after startup.
- netc: Wraps Textile network operations.
- isRunning: Boolean that is set to true if the initialization after calling Start finished successfully.
- Ready: A channel that gets emitted to after startup.
- cfg: A reference to the config object.
- isConnectedToHub: A boolean indicating if the initial Hub connection and authorization succeeded. If false, bucket operations will not be replicated on the Hub.

After initialization, Textile Client's state is mainly stored in the "meta thread". This meta thread stores a collection of the buckets the user has created or joined. This meta thread can be synced and joined in case the user wants to go cross-platform. The thread ID of the meta thread is stored in the local store. Operations over the meta thread are done in `collections.go`.

Creating and joining buckets (`bucket_factory.go`) adds a bucket instance to the meta thread. Listing and getting buckets query the meta thread to obtain the bucket's threadID and name. Using this info, these methods instantiate a Bucket object (`./bucket/bucket.go`), which exposes methods to do in bucket operations such as listing and adding files to a bucket.

## Hub authentication

Currently, we attempt to connect to the Hub on initialization. In coming releases, we might switch that so that it can be toggled on and off from the API. If the Hub connection succeeds, all bucket operations will include the auth token in the calls to Textile's bucket client. This will trigger Hub replication. The auth token is obtained by signing a challenge received from the Hub using the user private key. If the challenge is signed correctly, the Hub returns a non-expiring auth token that we store so that we don't need to re-authenticate.

The logic for authenticating and prepending the keys before bucket operations can be seen in `bucket_factory.go` in the method `getBucketContext`. It creates a Context instance that includes all the necessary information for accessing the correct thread and include the correct auth token.
