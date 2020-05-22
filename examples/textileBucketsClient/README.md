# textile in Go poc

Temporarily, buckets APIs aren't available in local threads so using the hub for all interactions. Once that is merged into threads, then we can use local one.

To get this working, the following is needed:

1. Run hub development environment by cloning `https://github.com/textileio/textile` and running `docker-compose -f docker-compose-dev.yml up --build`
2. Download the CLI at https://github.com/textileio/textile/releases
3. Run `install` from the bundle
4. `tt init --api=localhost:3006`
5. Hit http://127.0.0.1:8006/confirm/textilesession in the browser to "validate" email
6. Generate a "user" key `tt keys create` and choose user
7. Save values in `set-envs`
8. Build `go build .`
9. `source set-envs` to set environment variables
9. Run `./textileBucketsClient`

This will walk through the process of using the user key to generate a token, then using that against the hub to create bucket.