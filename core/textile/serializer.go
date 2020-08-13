package textile

import (
	"context"
	"encoding/json"
)

type StateBackup struct {
	MetathreadID string `json:"metathreadId"`
}

func (tc *textileClient) SerializeState(ctx context.Context) ([]byte, error) {
	mtID, err := tc.findOrCreateMetaThreadID(ctx)
	if err != nil {
		return nil, err
	}

	stringMtID := castDbIDToString(*mtID)

	backup := &StateBackup{
		MetathreadID: stringMtID,
	}

	return json.Marshal(backup)
}

func (tc *textileClient) RestoreState(ctx context.Context, state []byte) error {
	backup := &StateBackup{}
	if err := json.Unmarshal(state, backup); err != nil {
		return err
	}

	mtID, err := parseDbIDFromString(backup.MetathreadID)
	if err != nil {
		return err
	}

	mtIDInBytes := mtID.Bytes()
	return tc.store.Set([]byte(getThreadIDStoreKey(metaThreadName)), mtIDInBytes)
}
