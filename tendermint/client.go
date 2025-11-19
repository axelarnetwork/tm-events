package tendermint

import (
	"context"
	"fmt"
	"strings"

	"github.com/cometbft/cometbft/rpc/client"
	tmClient "github.com/cometbft/cometbft/rpc/client/http"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
)

// default client parameters
const (
	DefaultWSEndpoint = "/websocket"
	DefaultAddress    = "http://localhost:26657"
)

// StartClient connects a client to the given tendermint endpoint
func StartClient(address string, endpoint string) (client.Client, error) {
	if !validEndpoint(endpoint) {
		return nil, fmt.Errorf("invalid endpoint")
	}

	c, err := tmClient.New(address, endpoint)
	if err != nil {
		return nil, err
	}

	err = c.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to connect tendermint client at %s%s: %w", address, endpoint, err)
	}

	return c, nil
}

func validEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "/")
}

// RobustClient is a client that recreates the server connection of a connection failure occurs
type RobustClient struct {
	client       client.Client
	healthy      chan bool
	createClient func() (client.Client, error)
}

// BlockResults returns the all results of the block of given height
func (r *RobustClient) BlockResults(ctx context.Context, height *int64) (results *coretypes.ResultBlockResults, err error) {
	err = r.execute(func() error {
		results, err = r.client.BlockResults(ctx, height)
		return err
	})

	return results, err
}

// LatestSyncInfo returns the sync info of the node
func (r *RobustClient) LatestSyncInfo(ctx context.Context) (*coretypes.SyncInfo, error) {
	var status *coretypes.ResultStatus
	var err error
	err = r.execute(func() error {
		status, err = r.client.Status(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}

	return &status.SyncInfo, nil
}

// Stop stops the client and closes the server connection
func (r *RobustClient) Stop() error {
	return r.execute(func() error { return r.client.Stop() })
}

// Subscribe subscribes to the given query
func (r *RobustClient) Subscribe(ctx context.Context, subscriber, query string, outCapacity ...int) (out <-chan coretypes.ResultEvent, err error) {
	err = r.execute(func() error {
		out, err = r.client.Subscribe(ctx, subscriber, query, outCapacity...)
		return err
	})

	return out, err
}

// Unsubscribe unsubsribes the given query, if a subscription exists
func (r *RobustClient) Unsubscribe(ctx context.Context, subscriber, query string) error {
	return r.execute(func() error { return r.client.Unsubscribe(ctx, subscriber, query) })
}

// NewRobustClient returns a new RobustClient instance
func NewRobustClient(factory func() (client.Client, error)) *RobustClient {
	healthy := make(chan bool, 1)
	healthy <- false

	return &RobustClient{createClient: factory, healthy: healthy}
}

func (r *RobustClient) execute(f func() error) error {
	err := r.resetClientIfUnhealthy()
	if err != nil {
		return err
	}

	err = f()

	if err != nil && strings.Contains(err.Error(), "post failed") {
		healthy := <-r.healthy
		if healthy { // no other call has set it to false yet
			_ = r.client.Stop() // best effort to stop old client
		}
		r.healthy <- false
	}

	return err
}

func (r *RobustClient) resetClientIfUnhealthy() error {
	healthy := <-r.healthy
	if !healthy {
		cl, err := r.createClient()
		if err != nil {
			r.healthy <- false
			return err
		}
		r.client = cl
	}

	r.healthy <- true
	return nil
}
