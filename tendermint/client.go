package tendermint

import (
	"fmt"
	"strings"

	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client"
	tmClient "github.com/tendermint/tendermint/rpc/client/http"
)

// StartClient connects a client to the given tendermint endpoint
func StartClient(address string, endpoint string, logger tmlog.Logger) (client.Client, error) {
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
	logger.Info("Connected tendermint client at " + address)

	return c, nil
}

func validEndpoint(ep string) bool {
	return strings.HasPrefix(ep, "/")
}
