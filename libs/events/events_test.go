package events

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/log"
)

// TestAddListenerForEventFireOnce sets up an EventSwitch, subscribes a single
// listener to an event, and sends a string "data".
func TestAddListenerForEventFireOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	messages := make(chan EventData)
	require.NoError(t, evsw.AddListenerForEvent("listener", "event",
		func(ctx context.Context, data EventData) error {
			// test there's no deadlock if we remove the listener inside a callback
			evsw.RemoveListener("listener")
			select {
			case messages <- data:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	go evsw.FireEvent(ctx, "event", "data")
	received := <-messages
	if received != "data" {
		t.Errorf("message received does not match: %v", received)
	}
}

// TestAddListenerForEventFireMany sets up an EventSwitch, subscribes a single
// listener to an event, and sends a thousand integers.
func TestAddListenerForEventFireMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	doneSum := make(chan uint64)
	doneSending := make(chan uint64)
	numbers := make(chan uint64, 4)
	// subscribe one listener for one event
	require.NoError(t, evsw.AddListenerForEvent("listener", "event",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	// collect received events
	go sumReceivedNumbers(numbers, doneSum)
	// go fire events
	go fireEvents(ctx, evsw, "event", doneSending, uint64(1))
	checkSum := <-doneSending
	close(numbers)
	eventSum := <-doneSum
	if checkSum != eventSum {
		t.Errorf("not all messages sent were received.\n")
	}
}

// TestAddListenerForDifferentEvents sets up an EventSwitch, subscribes a single
// listener to three different events and sends a thousand integers for each
// of the three events.
func TestAddListenerForDifferentEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	doneSum := make(chan uint64)
	doneSending1 := make(chan uint64)
	doneSending2 := make(chan uint64)
	doneSending3 := make(chan uint64)
	numbers := make(chan uint64, 4)
	// subscribe one listener to three events
	require.NoError(t, evsw.AddListenerForEvent("listener", "event1",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener", "event3",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	// collect received events
	go sumReceivedNumbers(numbers, doneSum)
	// go fire events
	go fireEvents(ctx, evsw, "event1", doneSending1, uint64(1))
	go fireEvents(ctx, evsw, "event2", doneSending2, uint64(1))
	go fireEvents(ctx, evsw, "event3", doneSending3, uint64(1))
	var checkSum uint64
	checkSum += <-doneSending1
	checkSum += <-doneSending2
	checkSum += <-doneSending3
	close(numbers)
	eventSum := <-doneSum
	if checkSum != eventSum {
		t.Errorf("not all messages sent were received.\n")
	}
}

// TestAddDifferentListenerForDifferentEvents sets up an EventSwitch,
// subscribes a first listener to three events, and subscribes a second
// listener to two of those three events, and then sends a thousand integers
// for each of the three events.
func TestAddDifferentListenerForDifferentEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))

	t.Cleanup(evsw.Wait)

	doneSum1 := make(chan uint64)
	doneSum2 := make(chan uint64)
	doneSending1 := make(chan uint64)
	doneSending2 := make(chan uint64)
	doneSending3 := make(chan uint64)
	numbers1 := make(chan uint64, 4)
	numbers2 := make(chan uint64, 4)
	// subscribe two listener to three events
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event1",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event3",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener2", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener2", "event3",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	// collect received events for listener1
	go sumReceivedNumbers(numbers1, doneSum1)
	// collect received events for listener2
	go sumReceivedNumbers(numbers2, doneSum2)
	// go fire events
	go fireEvents(ctx, evsw, "event1", doneSending1, uint64(1))
	go fireEvents(ctx, evsw, "event2", doneSending2, uint64(1001))
	go fireEvents(ctx, evsw, "event3", doneSending3, uint64(2001))
	checkSumEvent1 := <-doneSending1
	checkSumEvent2 := <-doneSending2
	checkSumEvent3 := <-doneSending3
	checkSum1 := checkSumEvent1 + checkSumEvent2 + checkSumEvent3
	checkSum2 := checkSumEvent2 + checkSumEvent3
	close(numbers1)
	close(numbers2)
	eventSum1 := <-doneSum1
	eventSum2 := <-doneSum2
	if checkSum1 != eventSum1 ||
		checkSum2 != eventSum2 {
		t.Errorf("not all messages sent were received for different listeners to different events.\n")
	}
}

func TestAddAndRemoveListenerConcurrency(t *testing.T) {
	var (
		stopInputEvent = false
		roundCount     = 2000
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	done1 := make(chan struct{})
	done2 := make(chan struct{})

	// Must be executed concurrently to uncover the data race.
	// 1. RemoveListener
	go func() {
		defer close(done1)
		for i := 0; i < roundCount; i++ {
			evsw.RemoveListener("listener")
		}
	}()

	// 2. AddListenerForEvent
	go func() {
		defer close(done2)
		for i := 0; i < roundCount; i++ {
			index := i
			// we explicitly ignore errors here, since the listener will sometimes be removed
			// (that's what we're testing)
			_ = evsw.AddListenerForEvent("listener", fmt.Sprintf("event%d", index),
				func(ctx context.Context, data EventData) error {
					t.Errorf("should not run callback for %d.\n", index)
					stopInputEvent = true
					return nil
				})
		}
	}()

	<-done1
	<-done2

	evsw.RemoveListener("listener") // remove the last listener

	for i := 0; i < roundCount && !stopInputEvent; i++ {
		evsw.FireEvent(ctx, fmt.Sprintf("event%d", i), uint64(1001))
	}
}

// TestAddAndRemoveListener sets up an EventSwitch, subscribes a listener to
// two events, fires a thousand integers for the first event, then unsubscribes
// the listener and fires a thousand integers for the second event.
func TestAddAndRemoveListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	doneSum1 := make(chan uint64)
	doneSum2 := make(chan uint64)
	doneSending1 := make(chan uint64)
	doneSending2 := make(chan uint64)
	numbers1 := make(chan uint64, 4)
	numbers2 := make(chan uint64, 4)
	// subscribe two listener to three events
	require.NoError(t, evsw.AddListenerForEvent("listener", "event1",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	// collect received events for event1
	go sumReceivedNumbers(numbers1, doneSum1)
	// collect received events for event2
	go sumReceivedNumbers(numbers2, doneSum2)
	// go fire events
	go fireEvents(ctx, evsw, "event1", doneSending1, uint64(1))
	checkSumEvent1 := <-doneSending1
	// after sending all event1, unsubscribe for all events
	evsw.RemoveListener("listener")
	go fireEvents(ctx, evsw, "event2", doneSending2, uint64(1001))
	checkSumEvent2 := <-doneSending2
	close(numbers1)
	close(numbers2)
	eventSum1 := <-doneSum1
	eventSum2 := <-doneSum2
	if checkSumEvent1 != eventSum1 ||
		// correct value asserted by preceding tests, suffices to be non-zero
		checkSumEvent2 == uint64(0) ||
		eventSum2 != uint64(0) {
		t.Errorf("not all messages sent were received or unsubscription did not register.\n")
	}
}

// TestRemoveListener does basic tests on adding and removing
func TestRemoveListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	count := 10
	sum1, sum2 := 0, 0
	// add some listeners and make sure they work
	require.NoError(t, evsw.AddListenerForEvent("listener", "event1",
		func(ctx context.Context, data EventData) error {
			sum1++
			return nil
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener", "event2",
		func(ctx context.Context, data EventData) error {
			sum2++
			return nil
		}))

	for i := 0; i < count; i++ {
		evsw.FireEvent(ctx, "event1", true)
		evsw.FireEvent(ctx, "event2", true)
	}
	assert.Equal(t, count, sum1)
	assert.Equal(t, count, sum2)

	// remove one by event and make sure it is gone
	evsw.RemoveListenerForEvent("event2", "listener")
	for i := 0; i < count; i++ {
		evsw.FireEvent(ctx, "event1", true)
		evsw.FireEvent(ctx, "event2", true)
	}
	assert.Equal(t, count*2, sum1)
	assert.Equal(t, count, sum2)

	// remove the listener entirely and make sure both gone
	evsw.RemoveListener("listener")
	for i := 0; i < count; i++ {
		evsw.FireEvent(ctx, "event1", true)
		evsw.FireEvent(ctx, "event2", true)
	}
	assert.Equal(t, count*2, sum1)
	assert.Equal(t, count, sum2)
}

// TestAddAndRemoveListenersAsync sets up an EventSwitch, subscribes two
// listeners to three events, and fires a thousand integers for each event.
// These two listeners serve as the baseline validation while other listeners
// are randomly subscribed and unsubscribed.
// More precisely it randomly subscribes new listeners (different from the first
// two listeners) to one of these three events. At the same time it starts
// randomly unsubscribing these additional listeners from all events they are
// at that point subscribed to.
// NOTE: it is important to run this test with race conditions tracking on,
// `go test -race`, to examine for possible race conditions.
func TestRemoveListenersAsync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	evsw := NewEventSwitch(log.TestingLogger())
	require.NoError(t, evsw.Start(ctx))
	t.Cleanup(evsw.Wait)

	doneSum1 := make(chan uint64)
	doneSum2 := make(chan uint64)
	doneSending1 := make(chan uint64)
	doneSending2 := make(chan uint64)
	doneSending3 := make(chan uint64)
	numbers1 := make(chan uint64, 4)
	numbers2 := make(chan uint64, 4)
	// subscribe two listener to three events
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event1",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener1", "event3",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers1 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener2", "event1",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener2", "event2",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	require.NoError(t, evsw.AddListenerForEvent("listener2", "event3",
		func(ctx context.Context, data EventData) error {
			select {
			case numbers2 <- data.(uint64):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}))
	// collect received events for event1
	go sumReceivedNumbers(numbers1, doneSum1)
	// collect received events for event2
	go sumReceivedNumbers(numbers2, doneSum2)
	addListenersStress := func() {
		r1 := rand.New(rand.NewSource(time.Now().Unix()))
		r1.Seed(time.Now().UnixNano())
		for k := uint16(0); k < 400; k++ {
			listenerNumber := r1.Intn(100) + 3
			eventNumber := r1.Intn(3) + 1
			go evsw.AddListenerForEvent(fmt.Sprintf("listener%v", listenerNumber), //nolint:errcheck // ignore for tests
				fmt.Sprintf("event%v", eventNumber),
				func(context.Context, EventData) error { return nil })
		}
	}
	removeListenersStress := func() {
		r2 := rand.New(rand.NewSource(time.Now().Unix()))
		r2.Seed(time.Now().UnixNano())
		for k := uint16(0); k < 80; k++ {
			listenerNumber := r2.Intn(100) + 3
			go evsw.RemoveListener(fmt.Sprintf("listener%v", listenerNumber))
		}
	}
	addListenersStress()
	// go fire events
	go fireEvents(ctx, evsw, "event1", doneSending1, uint64(1))
	removeListenersStress()
	go fireEvents(ctx, evsw, "event2", doneSending2, uint64(1001))
	go fireEvents(ctx, evsw, "event3", doneSending3, uint64(2001))
	checkSumEvent1 := <-doneSending1
	checkSumEvent2 := <-doneSending2
	checkSumEvent3 := <-doneSending3
	checkSum := checkSumEvent1 + checkSumEvent2 + checkSumEvent3
	close(numbers1)
	close(numbers2)
	eventSum1 := <-doneSum1
	eventSum2 := <-doneSum2
	if checkSum != eventSum1 ||
		checkSum != eventSum2 {
		t.Errorf("not all messages sent were received.\n")
	}
}

//------------------------------------------------------------------------------
// Helper functions

// sumReceivedNumbers takes two channels and adds all numbers received
// until the receiving channel `numbers` is closed; it then sends the sum
// on `doneSum` and closes that channel.  Expected to be run in a go-routine.
func sumReceivedNumbers(numbers, doneSum chan uint64) {
	var sum uint64
	for {
		j, more := <-numbers
		sum += j
		if !more {
			doneSum <- sum
			close(doneSum)
			return
		}
	}
}

// fireEvents takes an EventSwitch and fires a thousand integers under
// a given `event` with the integers mootonically increasing from `offset`
// to `offset` + 999.  It additionally returns the addition of all integers
// sent on `doneChan` for assertion that all events have been sent, and enabling
// the test to assert all events have also been received.
func fireEvents(ctx context.Context, evsw Fireable, event string, doneChan chan uint64, offset uint64) {
	defer close(doneChan)

	var sentSum uint64
	for i := offset; i <= offset+uint64(999); i++ {
		if ctx.Err() != nil {
			break
		}

		evsw.FireEvent(ctx, event, i)
		sentSum += i
	}

	select {
	case <-ctx.Done():
	case doneChan <- sentSum:
	}
}
