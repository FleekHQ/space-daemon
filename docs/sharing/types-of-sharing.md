# Types of Sharing

Sharing happens around buckets. A bucket holds the file structure and pointers to each file. For each bucket, we use an additional Textile thread. This thread holds meta information around the bucket that's needed for sharing. In this doc you can read about the different types of sharing we support.

## Bucket sharing

The most simple sharing type. When you create a bucket (using the `CreateBucket` gRPC endpoint), a bucket with a single member, the creator of the bucket, will be created. If you use the `ShareBucket` gRPC, you can add all members you want. This is very similar to creating a team, or creating a channel in Slack.

## Select Group Sharing

This is analogue to a conversation between a set of people in Slack. When calling the `ShareItemsToSelectGroup` request, Space Daemon creates a bucket with a predefined slot for each member invited. When each member joins, the slots get filled with their public key. If you want to later on add another participant, you need to create a new bucket and either fork the "conversation" or start from scratch.

For more info, refer to the following diagram:

![sequence diagram](https://github.com/FleekHQ/space-daemon/blob/master/docs/sharing/select-group-sharing.png)
