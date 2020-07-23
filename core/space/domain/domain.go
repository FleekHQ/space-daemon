package domain

import (
	"time"

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

type FileSharingInfo struct {
	Bucket            string
	Path              string
	SharedFileCid     string
	SharedFileKey     string
	SpaceDownloadLink string
}

type Invitation struct {
	CustomMessage    string    `json:"customMessage"`
	InvitationID     string    `json:"invitationID"`
	InviteePublicKey string    `json:"inviteePublicKey"`
	InviterPublicKey string    `json:"inviterPublicKey"`
	Joined           bool      `json:"joined"`
	Read             bool      `json:"read"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Member struct {
	ID               core.InstanceID `json:"_id"`
	PublicKey        string          `json:"publicKey"`
	IsOwner          bool            `json:"isOwner"`
	InvitationID     string          `json:"invitationID"`
	InviterPublicKey string          `json:"inviterPublicKey"`
	CreatedAt        time.Time       `json:"createdAt"`
	Joined           bool            `json:"joined"`
	Read             bool            `json:"read"`
	CustomMessage    string          `json:"customMessage"`
}
