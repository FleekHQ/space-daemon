package common

import "context"

// NewBucketEncryptionKeyContext adds the encryption key to the context
// which is used to encrypt and decrypt requests to buckets client.
func NewBucketEncryptionKeyContext(ctx context.Context, key []byte) context.Context {
	if key == nil || len(key) == 0 {
		return ctx
	}
	return context.WithValue(ctx, "bucketEncryptionKey", key)
}

// BucketEncryptionKeyFromContext returns the bucket encryption key from a context.
func BucketEncryptionKeyFromContext(ctx context.Context) ([]byte, bool) {
	key, ok := ctx.Value("bucketEncryptionKey").([]byte)
	return key, ok
}
