## Powergate Daemon test

Tests interacts with the powergate daemon started by space daemon.
Before running the test, you need to do two things:

1. Startup the filecoin lotus local devnet. Run `make localnet-up` in the repositories root directory.
2. Space daemon need to be running with powergate started. Run `make && ./bin/space --filecoin` in this repos root directory.

### Run the test
1. Install modules with `yarn`.
2. Run test with `yarn test`.