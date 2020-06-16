package sync

import (
	"github.com/FleekHQ/space-poc/core/events"
)

type defaultNotifier struct {}

func (d defaultNotifier) SendFileEvent(event events.FileEvent) {
	return
}

func (d defaultNotifier) SendTextileEvent(event events.TextileEvent) {
	return
}
