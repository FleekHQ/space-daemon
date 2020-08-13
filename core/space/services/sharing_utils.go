package services

import (
	"github.com/FleekHQ/space-daemon/core/space/domain"
)

var EmptyFileSharingInfo = domain.FileSharingInfo{}

func generateFilesSharingZip() string {
	//return fmt.Sprintf("space_shared_files-%d.zip", time.Now().UnixNano())
	return "space_shared_files.zip"
}
