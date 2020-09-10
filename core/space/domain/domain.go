package domain

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

type Member struct {
	Address   string `json:"address"`
	PublicKey string `json:"publicKey"`
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

type NotificationTypes int

const (
	INVITATION NotificationTypes = iota
	USAGEALERT
	INVITATION_REPLY
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
	Keys             [][]byte         `json:""keys`
}

type InvitationReply struct {
	InvitationID string `json:"invitationID"`
}

type UsageAlert struct {
	Used    int64  `json:"used"`
	Limit   int64  `json:"limit"`
	Message string `json:message`
}

type MessageBody struct {
	Type NotificationTypes `json:"type"`
	Body []byte            `json:"body`
}

type Notification struct {
	ID               string            `json:"id"`
	Subject          string            `json:"subject"`
	Body             string            `json:"body"`
	NotificationType NotificationTypes `json:"notificationType"`
	CreatedAt        int64             `json:"createdAt"`
	ReadAt           int64             `json:"readAt"`
	// QUESTION: is there a way to enforce that only one of the below is present
	InvitationValue       Invitation      `json:"invitationValue"`
	UsageAlertValue       UsageAlert      `json:"usageAlertValue"`
	InvitationAcceptValue InvitationReply `json:"invitationAcceptValue"`
	RelatedObject         interface{}     `json:"relatedObject`
}

type APISessionTokens struct {
	HubToken      string
	ServicesToken string
}

type MirrorFile struct {
	Path       string
	BucketSlug string
	Backup     bool
	Shared     bool
}

type SharedDirEntry struct {
	DbID   string
	Bucket string
	FileInfo
	Members []Member // XXX: it is duplicated from FileInfo
}
