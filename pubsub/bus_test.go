package pubsub_test

import (
	"os"
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pubsub2 "github.com/axelarnetwork/tm-events/pubsub"
)

const (
	defaultDelayThreadStart = time.Millisecond * 6
)

// AfterThreadStart waits for the duration of delay thread start
func AfterThreadStart(t *testing.T) <-chan time.Time {
	return time.After(delayThreadStart(t))
}

// SleepForThreadStart pass go routine for the duration of delay thread start
func SleepForThreadStart(t *testing.T) {
	time.Sleep(delayThreadStart(t))
}

func delayThreadStart(t *testing.T) time.Duration {
	if val := os.Getenv("TEST_DELAY_THREAD_START"); val != "" {
		d, err := time.ParseDuration(val)
		require.NoError(t, err)

		return d
	}

	return defaultDelayThreadStart
}

func TestBus(t *testing.T) {
	bus := pubsub2.NewBus[crypto.Address]()
	defer bus.Close()

	ev := ed25519.GenPrivKey().PubKey().Address()

	assert.NoError(t, bus.Publish(ev))

	sub1 := bus.Subscribe(func(crypto.Address) bool { return true })
	sub2 := bus.Subscribe(func(crypto.Address) bool { return true })

	assert.NoError(t, bus.Publish(ev))

	select {
	case newEv := <-sub1:
		assert.Equal(t, ev, newEv)
	case <-AfterThreadStart(t):
		require.Fail(t, "time out")
	}

	select {
	case newEv := <-sub2:
		assert.Equal(t, ev, newEv)
	case <-AfterThreadStart(t):
		require.Fail(t, "time out")
	}

	assert.NoError(t, bus.Publish(ev))

	select {
	case newEv := <-sub1:
		assert.Equal(t, ev, newEv)
	case <-AfterThreadStart(t):
		require.Fail(t, "time out")
	}

	select {
	case newEv := <-sub2:
		assert.Equal(t, ev, newEv)
	case <-AfterThreadStart(t):
		require.Fail(t, "time out")
	}

	bus.Close()

	assert.Equal(t, pubsub2.ErrNotRunning, bus.Publish(ev))
}
