package domain

type AppConfig struct {
	FolderPath string
	Port       int
	AppPath    string
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

type PathInfo struct {
	Path     string
	IpfsHash string
	IsDir    bool
}

type KeyPair struct {
	PublicKey  string
	PrivateKey string
}
