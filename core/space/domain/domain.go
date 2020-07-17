package domain

import (
	core "github.com/textileio/go-threads/core/db"
)

type AppConfig struct {
	Port                 int
	AppPath              string
	TextileHubTarget     string
	TextileThreadsTarget string
}

type DirEntry struct {
	Path          string
	IsDir         bool
	Name          string
	SizeInBytes   string
	Created       string
	Updated       string
	FileExtension string
}

type ThreadInfo struct {
	Addresses []string
	Key       string
}

type FileInfo struct {
	DirEntry
	IpfsHash string
}

type OpenFileInfo struct {
	Location string
}

type KeyPair struct {
	PublicKey  string
	PrivateKey string
}

type AddItemResult struct {
	SourcePath string
	BucketPath string
	Bytes      int64
	Error      error
}

type AddItemsResponse struct {
	TotalFiles int64
	TotalBytes int64
	Error      error
}

type AddWatchFile struct {
	LocalPath  string `json:"local_path"`
	BucketPath string `json:"bucket_path"`
	BucketKey  string `json:"bucket_key"`
}

type Identity struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
	Username  string `json:"username"`
}

type APIError struct {
	Message string `json:"message"`
}

type InvitationType int

const (
	INVITE_THROUGH_EMAIL InvitationType = iota
	INVITE_THROUGH_ADDRESS
)

type Invitation struct {
	InvitationType  InvitationType `json:"invitationType"`
	InvitationValue string         `json:"invitationValue"`
}

// this is to be a singleton, just one record
// to store metadata about that bucket inside
// the buckets thread
type BucketThreadMeta struct {
	ID                  core.InstanceID `json:"_id"`
	IsSelectGroupBucket bool            `json:isSelectGroupBucket`
}

type Member struct {
	ID             core.InstanceID `json:"_id"`
	Address        string          `json:"address"`
	PublicKey      string          `json:"publicKey"`
	Username       string          `json:"username"`
	Email          string          `json:"email"`
	IsOwner        bool            `json:"isOwner"`
	InvitationID   string          `json:"invitationID"`
	InvitationType InvitationType  `json:"invitationType"`
}
