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
	Error      error
}

type AddWatchFile struct {
	LocalPath  string
	BucketPath string
	BucketKey  string
}
