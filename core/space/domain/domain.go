package domain

import "fmt"

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
	Members       []Member
}

type ThreadInfo struct {
	Addresses []string
	Key       string
}

type FileInfo struct {
	DirEntry
	IpfsHash          string
	BackedUp          bool
	LocallyAvailable  bool
	BackupInProgress  bool
	RestoreInProgress bool
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

type Member struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
}

type AddWatchFile struct {
	DbId       string `json:"dbId"`
	LocalPath  string `json:"local_path"`
	BucketPath string `json:"bucket_path"`
	BucketKey  string `json:"bucket_key"`
	BucketSlug string `json:"bucket_slug"`
	IsRemote   bool   `json:"isRemote"`
	Cid        string `json:"cid"`
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

type NotificationTypes int

const (
	UNKNOWN NotificationTypes = iota
	INVITATION
	USAGEALERT
	INVITATION_REPLY
	REVOKED_INVITATION
)

type FullPath struct {
	DbId      string `json:"dbId"`
	BucketKey string `json:"bucketKey"`
	Bucket    string `json:"bucket"`
	Path      string `json:"path"`
}

type InvitationStatus int

const (
	PENDING  InvitationStatus = 0
	ACCEPTED InvitationStatus = 1
	REJECTED InvitationStatus = 2
)

type Invitation struct {
	InviterPublicKey string           `json:"inviterPublicKey"`
	InviteePublicKey string           `json:"inviteePublicKey"`
	InvitationID     string           `json:"invitationID"`
	Status           InvitationStatus `json:"status"`
	ItemPaths        []FullPath       `json:"itemPaths"`
	Keys             [][]byte         `json:"keys"`
}

type InvitationReply struct {
	InvitationID string `json:"invitationID"`
}

// Represents when an inviter unshared access to previously shared files in ItemPaths
type RevokedInvitation struct {
	InviterPublicKey string     `json:"inviterPublicKey"`
	InviteePublicKey string     `json:"inviteePublicKey"`
	ItemPaths        []FullPath `json:"itemPaths"`
	Keys             [][]byte   `json:"keys"`
}

type UsageAlert struct {
	Used    int64  `json:"used"`
	Limit   int64  `json:"limit"`
	Message string `json:"message"`
}

type MessageBody struct {
	Type NotificationTypes `json:"type"`
	Body []byte            `json:"body"`
}

type Notification struct {
	ID               string            `json:"id"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	NotificationType NotificationTypes `json:"notificationType"`
	CreatedAt        int64             `json:"createdAt"`
	ReadAt           int64             `json:"readAt"`
	// QUESTION: is there a way to enforce that only one of the below is present
	InvitationValue        Invitation        `json:"invitationValue"`
	UsageAlertValue        UsageAlert        `json:"usageAlertValue"`
	InvitationAcceptValue  InvitationReply   `json:"invitationAcceptValue"`
	RevokedInvitationValue RevokedInvitation `json:"revokedInvitationValue"`
	RelatedObject          interface{}       `json:"relatedObject"`
}

type APISessionTokens struct {
	HubToken      string
	ServicesToken string
}

type MirrorFile struct {
	Path              string
	BucketSlug        string
	Backup            bool
	Shared            bool
	BackupInProgress  bool
	RestoreInProgress bool
}

type SharedDirEntry struct {
	DbID         string
	Bucket       string
	IsPublicLink bool
	FileInfo
	Members []Member // XXX: it is duplicated from FileInfo
}

type SearchFileEntry struct {
	FileInfo
	Bucket string
	DbID   string
}

type KeyBackupType int

const (
	PASSWORD KeyBackupType = 0
	GOOGLE   KeyBackupType = 1
	TWITTER  KeyBackupType = 2
	EMAIL    KeyBackupType = 3
)

func (b KeyBackupType) String() string {
	switch b {
	case 0:
		return "password"
	case 1:
		return "google"
	case 2:
		return "twitter"
	case 3:
		return "email"
	default:
		return fmt.Sprintf("%d", int(b))
	}
}

// SharedFilesRoleAction represents action to be performed on the role
type SharedFilesRoleAction int

const (
	DeleteRoleAction SharedFilesRoleAction = iota
	ReadWriteRoleAction
)
