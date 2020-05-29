package domain

type AppConfig struct {
	FolderPath string
	Port       int
	AppPath    string
}

type DirEntry struct {
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
