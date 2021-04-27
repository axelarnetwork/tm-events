package client

import (
	"fmt"
	"github.com/axelarnetwork/tm-events/pkg/errors"
	tmClient "github.com/tendermint/tendermint/rpc/client/http"
	"strings"
)

type RPCClient struct {
	*tmClient.HTTP
}

const (
	DefaultWSEndpoint = "/websocket"
	DefaultAddress    = "http://localhost:26657"
)

func NewClient(address string, endpoint string) (*RPCClient, error) {
	wrap := errors.Wrapper("failed to create tendermint client")

	if !validEndpoint(endpoint) {
		return nil, wrap(fmt.Errorf("invalid endpoint"))
	}

	c, err := tmClient.New(address, endpoint)
	if err != nil {
		return nil, wrap(err)
	}

	err = c.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to connect tendermint client at %s%s: %w", address, endpoint, err)
	}
	fmt.Printf("Connected tendermint client at %s\n", address)

	return &RPCClient{c}, nil
}

func validEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "/")
}
