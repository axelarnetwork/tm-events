package mock

import "github.com/cometbft/cometbft/rpc/client"

//go:generate moq -pkg mock -out ./types.go  . Client

// Client alias for mocking
type Client client.Client
