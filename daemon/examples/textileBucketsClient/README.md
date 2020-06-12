# textile in Go poc

Temporarily, buckets APIs aren't available in local threads so using the hub for all interactions. Once that is merged into threads, then we can use local one.

To run the in-process `threadsd` example:
1. Build `go build .`
2. Run `./textileBucketsClient threads`

To run the bucket creation with hub example: 
1. Set the host, key and secret in set-envs. Use textile-hub-dev.fleek.co for the host. Key and secret should be the one shared in Slack or generated using the `tt` cli (see extra notes below)
2. `source set-envs`
3. `go build .`
4. `./textileBucketsClient hub`

### Textile CLI

1. Download bundle at https://github.com/textileio/textile/releases
2. Extract
3. Go into extracted folder and run `./install`
4. Run `tt init --api=textile-hub-dev.fleek.co:3006`
5. When asked for email validation, hit http://textile-hub-dev.fleek.co:8006/confirm/textilesession to auto validate
6. Run `tt keys create --api=textile-hub-dev.fleek.co:3006` to create keys.
7. You can also use all the other `tt` commands pointing to our dev hub by adding the flag `--api=textile-hub-dev.fleek.co:3006`