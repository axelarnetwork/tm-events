package mock

import (
	tmpubsub "github.com/cometbft/cometbft/libs/pubsub"
)

//go:generate moq -pkg mock -out ./imported.go . Query

type (
	// Query interface alias for mocking
	Query tmpubsub.Query
)
