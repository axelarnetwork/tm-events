package tendermint_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/rpc/client"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/axelarnetwork/tm-events/tendermint"
	"github.com/axelarnetwork/tm-events/tendermint/mock"
	. "github.com/axelarnetwork/utils/test"
	"github.com/axelarnetwork/utils/test/rand"
)

func TestResettableClient(t *testing.T) {
	var (
		expectedBlockHeight = rand.PosI64()
		resettableClient    *tendermint.RobustClient
	)
	Given("a resettable client with an unreliable connection", func(t *testing.T) {
		resettableClient = tendermint.NewRobustClient(func() (client.Client, error) {
			untilFailure := rand.I64Between(1, 100)
			calls := int64(0)
			return &mock.ClientMock{
				ABCIInfoFunc: func(context.Context) (*coretypes.ResultABCIInfo, error) {
					calls++
					if calls > untilFailure {
						return nil, fmt.Errorf("post failed: some connection error")
					}

					return &coretypes.ResultABCIInfo{
						Response: types.ResponseInfo{LastBlockHeight: expectedBlockHeight},
					}, nil
				},
				StatusFunc: func(context.Context) (*coretypes.ResultStatus, error) {
					calls++
					if calls > untilFailure {
						return nil, fmt.Errorf("post failed: some connection error")
					}

					return &coretypes.ResultStatus{
						SyncInfo: coretypes.SyncInfo{LatestBlockHeight: expectedBlockHeight},
					}, nil
				},
				StopFunc: func() error { return nil },
			}, nil
		})
	}).
		When("calling a tendermint function until the connection fails", func(t *testing.T) {
			var err error
			for err == nil {
				var blockHeight int64
				blockHeight, err = resettableClient.LatestBlockHeight(context.Background())
				if err == nil {
					assert.Equal(t, expectedBlockHeight, blockHeight)
				}
			}

			err = nil
			for err == nil {
				var blockHeight int64
				blockHeight, err = resettableClient.LatestNodeBlockHeight(context.Background())
				if err == nil {
					assert.Equal(t, expectedBlockHeight, blockHeight)
				}
			}
		}).
		Then("recover the connection", func(t *testing.T) {
			blockHeight, err := resettableClient.LatestBlockHeight(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, expectedBlockHeight, blockHeight)

			blockHeight, err = resettableClient.LatestNodeBlockHeight(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, expectedBlockHeight, blockHeight)
		}).Run(t, 20)
}
