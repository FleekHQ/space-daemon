package domain

import (
	"time"
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
	SharedFileCid     string
	SharedFileKey     string
	SpaceDownloadLink string
}

type FullPath struct {
	DbId   string `json:"dbId"`
	Bucket string `json:"bucket"`
	Path   string `json:"path"`
}

type InvitationStatus int

const (
	Pending InvitationStatus = 0
	Accepted
	Rejected
)

type MessageType int

const (
	InvitationMessage MessageType = 0
	UsageAlertMessage
)

type MessageBody struct {
	Type MessageType `json:"type"`
	Body interface{} `json:"body`
}

type Invitation struct {
	CustomMessage    string           `json:"customMessage"`
	InvitationID     string           `json:"invitationID"`
	InviteePublicKey string           `json:"inviteePublicKey"`
	InviterPublicKey string           `json:"inviterPublicKey"`
	Status           InvitationStatus `json:"status"`
	Paths            []FullPath       `json:"Paths"`
	ReadAt           time.Time        `json:"readAt"`
	CreatedAt        time.Time        `json:"createdAt"`
}
type APISessionTokens struct {
	HubToken      string
	ServicesToken string
}
