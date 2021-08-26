// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"github.com/axelarnetwork/tm-events/pubsub"
	"sync"
)

// Ensure, that QueryMock does implement Query.
// If this is not the case, regenerate this file with moq.
var _ Query = &QueryMock{}

// QueryMock is a mock implementation of Query.
//
// 	func TestSomethingThatUsesQuery(t *testing.T) {
//
// 		// make and configure a mocked Query
// 		mockedQuery := &QueryMock{
// 			MatchesFunc: func(events map[string][]string) (bool, error) {
// 				panic("mock out the Matches method")
// 			},
// 			StringFunc: func() string {
// 				panic("mock out the String method")
// 			},
// 		}
//
// 		// use mockedQuery in code that requires Query
// 		// and then make assertions.
//
// 	}
type QueryMock struct {
	// MatchesFunc mocks the Matches method.
	MatchesFunc func(events map[string][]string) (bool, error)

	// StringFunc mocks the String method.
	StringFunc func() string

	// calls tracks calls to the methods.
	calls struct {
		// Matches holds details about calls to the Matches method.
		Matches []struct {
			// Events is the events argument value.
			Events map[string][]string
		}
		// String holds details about calls to the String method.
		String []struct {
		}
	}
	lockMatches sync.RWMutex
	lockString  sync.RWMutex
}

// Matches calls MatchesFunc.
func (mock *QueryMock) Matches(events map[string][]string) (bool, error) {
	if mock.MatchesFunc == nil {
		panic("QueryMock.MatchesFunc: method is nil but Query.Matches was just called")
	}
	callInfo := struct {
		Events map[string][]string
	}{
		Events: events,
	}
	mock.lockMatches.Lock()
	mock.calls.Matches = append(mock.calls.Matches, callInfo)
	mock.lockMatches.Unlock()
	return mock.MatchesFunc(events)
}

// MatchesCalls gets all the calls that were made to Matches.
// Check the length with:
//     len(mockedQuery.MatchesCalls())
func (mock *QueryMock) MatchesCalls() []struct {
	Events map[string][]string
} {
	var calls []struct {
		Events map[string][]string
	}
	mock.lockMatches.RLock()
	calls = mock.calls.Matches
	mock.lockMatches.RUnlock()
	return calls
}

// String calls StringFunc.
func (mock *QueryMock) String() string {
	if mock.StringFunc == nil {
		panic("QueryMock.StringFunc: method is nil but Query.String was just called")
	}
	callInfo := struct {
	}{}
	mock.lockString.Lock()
	mock.calls.String = append(mock.calls.String, callInfo)
	mock.lockString.Unlock()
	return mock.StringFunc()
}

// StringCalls gets all the calls that were made to String.
// Check the length with:
//     len(mockedQuery.StringCalls())
func (mock *QueryMock) StringCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockString.RLock()
	calls = mock.calls.String
	mock.lockString.RUnlock()
	return calls
}

// Ensure, that BusMock does implement Bus.
// If this is not the case, regenerate this file with moq.
var _ Bus = &BusMock{}

// BusMock is a mock implementation of Bus.
//
// 	func TestSomethingThatUsesBus(t *testing.T) {
//
// 		// make and configure a mocked Bus
// 		mockedBus := &BusMock{
// 			CloseFunc: func()  {
// 				panic("mock out the Close method")
// 			},
// 			DoneFunc: func() <-chan struct{} {
// 				panic("mock out the Done method")
// 			},
// 			PublishFunc: func(event pubsub.Event) error {
// 				panic("mock out the Publish method")
// 			},
// 			SubscribeFunc: func() (pubsub.Subscriber, error) {
// 				panic("mock out the Subscribe method")
// 			},
// 		}
//
// 		// use mockedBus in code that requires Bus
// 		// and then make assertions.
//
// 	}
type BusMock struct {
	// CloseFunc mocks the Close method.
	CloseFunc func()

	// DoneFunc mocks the Done method.
	DoneFunc func() <-chan struct{}

	// PublishFunc mocks the Publish method.
	PublishFunc func(event pubsub.Event) error

	// SubscribeFunc mocks the Subscribe method.
	SubscribeFunc func() (pubsub.Subscriber, error)

	// calls tracks calls to the methods.
	calls struct {
		// Close holds details about calls to the Close method.
		Close []struct {
		}
		// Done holds details about calls to the Done method.
		Done []struct {
		}
		// Publish holds details about calls to the Publish method.
		Publish []struct {
			// Event is the event argument value.
			Event pubsub.Event
		}
		// Subscribe holds details about calls to the Subscribe method.
		Subscribe []struct {
		}
	}
	lockClose     sync.RWMutex
	lockDone      sync.RWMutex
	lockPublish   sync.RWMutex
	lockSubscribe sync.RWMutex
}

// Close calls CloseFunc.
func (mock *BusMock) Close() {
	if mock.CloseFunc == nil {
		panic("BusMock.CloseFunc: method is nil but Bus.Close was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	mock.CloseFunc()
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//     len(mockedBus.CloseCalls())
func (mock *BusMock) CloseCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// Done calls DoneFunc.
func (mock *BusMock) Done() <-chan struct{} {
	if mock.DoneFunc == nil {
		panic("BusMock.DoneFunc: method is nil but Bus.Done was just called")
	}
	callInfo := struct {
	}{}
	mock.lockDone.Lock()
	mock.calls.Done = append(mock.calls.Done, callInfo)
	mock.lockDone.Unlock()
	return mock.DoneFunc()
}

// DoneCalls gets all the calls that were made to Done.
// Check the length with:
//     len(mockedBus.DoneCalls())
func (mock *BusMock) DoneCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockDone.RLock()
	calls = mock.calls.Done
	mock.lockDone.RUnlock()
	return calls
}

// Publish calls PublishFunc.
func (mock *BusMock) Publish(event pubsub.Event) error {
	if mock.PublishFunc == nil {
		panic("BusMock.PublishFunc: method is nil but Bus.Publish was just called")
	}
	callInfo := struct {
		Event pubsub.Event
	}{
		Event: event,
	}
	mock.lockPublish.Lock()
	mock.calls.Publish = append(mock.calls.Publish, callInfo)
	mock.lockPublish.Unlock()
	return mock.PublishFunc(event)
}

// PublishCalls gets all the calls that were made to Publish.
// Check the length with:
//     len(mockedBus.PublishCalls())
func (mock *BusMock) PublishCalls() []struct {
	Event pubsub.Event
} {
	var calls []struct {
		Event pubsub.Event
	}
	mock.lockPublish.RLock()
	calls = mock.calls.Publish
	mock.lockPublish.RUnlock()
	return calls
}

// Subscribe calls SubscribeFunc.
func (mock *BusMock) Subscribe() (pubsub.Subscriber, error) {
	if mock.SubscribeFunc == nil {
		panic("BusMock.SubscribeFunc: method is nil but Bus.Subscribe was just called")
	}
	callInfo := struct {
	}{}
	mock.lockSubscribe.Lock()
	mock.calls.Subscribe = append(mock.calls.Subscribe, callInfo)
	mock.lockSubscribe.Unlock()
	return mock.SubscribeFunc()
}

// SubscribeCalls gets all the calls that were made to Subscribe.
// Check the length with:
//     len(mockedBus.SubscribeCalls())
func (mock *BusMock) SubscribeCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockSubscribe.RLock()
	calls = mock.calls.Subscribe
	mock.lockSubscribe.RUnlock()
	return calls
}

// Ensure, that SubscriberMock does implement Subscriber.
// If this is not the case, regenerate this file with moq.
var _ Subscriber = &SubscriberMock{}

// SubscriberMock is a mock implementation of Subscriber.
//
// 	func TestSomethingThatUsesSubscriber(t *testing.T) {
//
// 		// make and configure a mocked Subscriber
// 		mockedSubscriber := &SubscriberMock{
// 			CloneFunc: func() (pubsub.Subscriber, error) {
// 				panic("mock out the Clone method")
// 			},
// 			CloseFunc: func()  {
// 				panic("mock out the Close method")
// 			},
// 			DoneFunc: func() <-chan struct{} {
// 				panic("mock out the Done method")
// 			},
// 			EventsFunc: func() <-chan pubsub.Event {
// 				panic("mock out the Events method")
// 			},
// 		}
//
// 		// use mockedSubscriber in code that requires Subscriber
// 		// and then make assertions.
//
// 	}
type SubscriberMock struct {
	// CloneFunc mocks the Clone method.
	CloneFunc func() (pubsub.Subscriber, error)

	// CloseFunc mocks the Close method.
	CloseFunc func()

	// DoneFunc mocks the Done method.
	DoneFunc func() <-chan struct{}

	// EventsFunc mocks the Events method.
	EventsFunc func() <-chan pubsub.Event

	// calls tracks calls to the methods.
	calls struct {
		// Clone holds details about calls to the Clone method.
		Clone []struct {
		}
		// Close holds details about calls to the Close method.
		Close []struct {
		}
		// Done holds details about calls to the Done method.
		Done []struct {
		}
		// Events holds details about calls to the Events method.
		Events []struct {
		}
	}
	lockClone  sync.RWMutex
	lockClose  sync.RWMutex
	lockDone   sync.RWMutex
	lockEvents sync.RWMutex
}

// Clone calls CloneFunc.
func (mock *SubscriberMock) Clone() (pubsub.Subscriber, error) {
	if mock.CloneFunc == nil {
		panic("SubscriberMock.CloneFunc: method is nil but Subscriber.Clone was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClone.Lock()
	mock.calls.Clone = append(mock.calls.Clone, callInfo)
	mock.lockClone.Unlock()
	return mock.CloneFunc()
}

// CloneCalls gets all the calls that were made to Clone.
// Check the length with:
//     len(mockedSubscriber.CloneCalls())
func (mock *SubscriberMock) CloneCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClone.RLock()
	calls = mock.calls.Clone
	mock.lockClone.RUnlock()
	return calls
}

// Close calls CloseFunc.
func (mock *SubscriberMock) Close() {
	if mock.CloseFunc == nil {
		panic("SubscriberMock.CloseFunc: method is nil but Subscriber.Close was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	mock.CloseFunc()
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//     len(mockedSubscriber.CloseCalls())
func (mock *SubscriberMock) CloseCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// Done calls DoneFunc.
func (mock *SubscriberMock) Done() <-chan struct{} {
	if mock.DoneFunc == nil {
		panic("SubscriberMock.DoneFunc: method is nil but Subscriber.Done was just called")
	}
	callInfo := struct {
	}{}
	mock.lockDone.Lock()
	mock.calls.Done = append(mock.calls.Done, callInfo)
	mock.lockDone.Unlock()
	return mock.DoneFunc()
}

// DoneCalls gets all the calls that were made to Done.
// Check the length with:
//     len(mockedSubscriber.DoneCalls())
func (mock *SubscriberMock) DoneCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockDone.RLock()
	calls = mock.calls.Done
	mock.lockDone.RUnlock()
	return calls
}

// Events calls EventsFunc.
func (mock *SubscriberMock) Events() <-chan pubsub.Event {
	if mock.EventsFunc == nil {
		panic("SubscriberMock.EventsFunc: method is nil but Subscriber.Events was just called")
	}
	callInfo := struct {
	}{}
	mock.lockEvents.Lock()
	mock.calls.Events = append(mock.calls.Events, callInfo)
	mock.lockEvents.Unlock()
	return mock.EventsFunc()
}

// EventsCalls gets all the calls that were made to Events.
// Check the length with:
//     len(mockedSubscriber.EventsCalls())
func (mock *SubscriberMock) EventsCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockEvents.RLock()
	calls = mock.calls.Events
	mock.lockEvents.RUnlock()
	return calls
}
