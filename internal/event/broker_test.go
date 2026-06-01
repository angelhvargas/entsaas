package event

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

type testMessage struct {
	Text string `json:"text"`
}

func TestInMemoryBroker_PubSub(t *testing.T) {
	broker := NewInMemoryBroker()
	ctx := context.Background()

	var receivedText1, receivedText2 string
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Subscriber 1
	err := broker.Subscribe(ctx, "events.test", func(payload []byte) error {
		defer wg.Done()
		var msg testMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return err
		}
		receivedText1 = msg.Text
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe 1: %v", err)
	}

	// Subscriber 2
	err = broker.Subscribe(ctx, "events.test", func(payload []byte) error {
		defer wg.Done()
		var msg testMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			return err
		}
		receivedText2 = msg.Text
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe 2: %v", err)
	}

	// Publish
	err = broker.Publish(ctx, "events.test", testMessage{Text: "broadcast"})
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	// Wait for goroutines with a timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for subscribers to receive event")
	}

	if receivedText1 != "broadcast" {
		t.Errorf("subscriber 1 got %q, want %q", receivedText1, "broadcast")
	}
	if receivedText2 != "broadcast" {
		t.Errorf("subscriber 2 got %q, want %q", receivedText2, "broadcast")
	}
}
