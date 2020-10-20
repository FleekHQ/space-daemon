package notifier

import (
	"github.com/FleekHQ/space-daemon/core/textile/sync"
	"github.com/ipfs/interface-go-ipfs-core/path"
)

type Notifier struct {
	s sync.Synchronizer
}

func New(s sync.Synchronizer) *Notifier {
	return &Notifier{
		s: s,
	}
}

func (n *Notifier) OnUploadFile(bucketSlug string, bucketPath string, result path.Resolved, root path.Path) {
	n.s.NotifyItemAdded(bucketSlug, bucketPath)
}
