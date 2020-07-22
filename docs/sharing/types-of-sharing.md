# Types of Sharing

Sharing happens around buckets. A bucket holds the file structure and pointers to each file. For each bucket, we use an additional Textile thread. This thread holds meta information around the bucket that's needed for sharing. In this doc you can read about the different types of sharing we support.

## Bucket sharing

The most simple sharing type. When you create a bucket (using the `CreateBucket` gRPC endpoint), a bucket with a single member, the creator of the bucket, will be created. If you use the `ShareBucket` gRPC, you can add all members you want. This is very similar to creating a team, or creating a channel in Slack.

## Copy and share

This is analogue to a conversation between a set of people in Slack. When calling the `CopyAndShareFiles` request, Space Daemon creates a bucket with members access already predefined by the public keys given. If a bucket with the same set of public keys exists, it is used instead. Then, it copies the set of files over to that bucket and sends an invitation through Textile's Hub inboxing.

## Public File Sharing

When calling `GeneratePublicFileLink`, the file is going to be encrypted and uploaded to IPFS. The link will point to a gateway so that anyone with the decryption key will be able to download the file.

We are evaluating also creating a bucket around this single file, so that the link can also be used to join the bucket and modify the file collaboratively.
