package tendermint

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/types/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"
	tmClient "github.com/tendermint/tendermint/rpc/client/http"
)

type RPCClient struct {
	*tmClient.HTTP
	logger tmlog.Logger
}

const (
	DefaultWSEndpoint = "/websocket"
	DefaultAddress    = "http://localhost:26657"
)

func NewClient(address string, endpoint string, logger tmlog.Logger) (*RPCClient, error) {
	errDescr := "failed to create tendermint client"
	if !validEndpoint(endpoint) {
		return nil, errors.Wrap(fmt.Errorf("invalid endpoint"), errDescr)
	}

	c, err := tmClient.New(address, endpoint)
	if err != nil {
		return nil, errors.Wrap(err, errDescr)
	}

	err = c.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to connect tendermint client at %s%s: %w", address, endpoint, err)
	}
	logger.Info("Connected tendermint client at " + address)

	return &RPCClient{c, logger}, nil
}

func validEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "/")
}
