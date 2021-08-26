package events

import (
	"context"

	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

type blockClient struct{ rpcclient.Client }

func (b blockClient) LatestBlockHeight(ctx context.Context) (int64, error) {
	status, err := b.Status(ctx)
	if err != nil {
		return 0, err
	}
	return status.SyncInfo.LatestBlockHeight, nil
}

// NewBlockClient returns a new events.BlockClient instance
func NewBlockClient(client rpcclient.Client) BlockClient {
	return blockClient{client}
}
