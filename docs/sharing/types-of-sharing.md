# Types of Sharing

Sharing can happen at the file level or the bucket level.  Legacy sharing was done at the bucket level so those interfaces are left for continued usage.  Space app however will now rely on file level access control and sharing hence there will be a mixed set of interfaces for sharing.

In bucket sharing, you can share an entire bucket but getting the thread info to share.  A bucket holds the file structure and pointers to each file. For each bucket, we use an additional Textile thread. This thread holds meta information around the bucket that's needed for sharing. In this doc you can read about the different types of sharing we support.

In file level sharing, a mirror copy of the bucket is made in the hub and only that shared path will be added to that bucket and access controlled via Textile hub's new file level access control feature.

## Bucket sharing

The most simple sharing type. When you create a bucket (using the `CreateBucket` gRPC endpoint), a bucket with a single member, the creator of the bucket, will be created. If you use the `ShareBucket` gRPC, you can add all members you want. This is very similar to creating a team, or creating a channel in Slack.

## File level sharing

For this, a set of paths are shared and like previously described, a mirror bucket is created with just the paths that are being shared copied over.  Furthermore since these files will be on the hub, a Space encryption layer is added where a new key will be used for each file.  Finally the file specific key for the paths being shared will be sent via hub inboxing so that there is a way to retreive and decrypt files shared through a hub without exposing the content to the hub.

## Public File Sharing

When calling `GeneratePublicFileLink`, the file is going to be encrypted and uploaded to IPFS. The link will point to a gateway so that anyone with the decryption key will be able to download the file.

We are evaluating also creating a bucket around this single file, so that the link can also be used to join the bucket and modify the file collaboratively.
