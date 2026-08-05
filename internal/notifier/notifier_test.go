package notifier

import (
	"sync"
	"testing"
	"time"

	"github.com/koftamainee/ozon-bank-test-assigment/internal/domain"
)

func recv(t *testing.T, ch <-chan domain.Comment) domain.Comment {
	t.Helper()
	select {
	case c := <-ch:
		return c
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for comment")
		return domain.Comment{}
	}
}

func TestSubscribeReceivesPublished(t *testing.T) {
	b := New()
	ch, unsubscribe := b.Subscribe(1)
	defer unsubscribe()

	want := domain.Comment{ID: 7, PostID: 1, Body: "hello"}
	b.Publish(1, want)

	if got := recv(t, ch); got.ID != want.ID {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestPublishDeliversToMultipleSubscribers(t *testing.T) {
	b := New()
	ch1, unsubscribe1 := b.Subscribe(1)
	ch2, unsubscribe2 := b.Subscribe(1)
	defer unsubscribe1()
	defer unsubscribe2()

	b.Publish(1, domain.Comment{ID: 1})

	recv(t, ch1)
	recv(t, ch2)
}

func TestPublishOnlyDeliversToPostSubscribers(t *testing.T) {
	b := New()
	ch1, unsubscribe1 := b.Subscribe(1)
	ch2, unsubscribe2 := b.Subscribe(2)
	defer unsubscribe1()
	defer unsubscribe2()

	b.Publish(1, domain.Comment{ID: 1})

	recv(t, ch1)
	select {
	case c := <-ch2:
		t.Fatalf("post 2 must not receive post 1 comment, got %+v", c)
	default:
	}
}

func TestPublishDoesNotBlockOnSlowSubscriber(t *testing.T) {
	b := New()
	_, unsubscribe := b.Subscribe(1)
	defer unsubscribe()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			b.Publish(1, domain.Comment{ID: int64(i)})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on slow subscriber")
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	b := New()
	ch, unsubscribe := b.Subscribe(1)

	b.Publish(1, domain.Comment{ID: 1})
	recv(t, ch)

	unsubscribe()
	unsubscribe()

	b.Publish(1, domain.Comment{ID: 2})
	select {
	case c, ok := <-ch:
		if ok {
			t.Fatalf("unsubscribed subscriber must not receive, got %+v", c)
		}
	default:
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	b := New()
	ch, unsubscribe := b.Subscribe(1)

	b.Publish(1, domain.Comment{ID: 1})
	recv(t, ch)

	unsubscribe()

	c, ok := <-ch
	if ok {
		t.Fatalf("channel must be closed after unsubscribe, got %+v", c)
	}
}

func TestPublishToPostWithoutSubscribers(t *testing.T) {
	b := New()
	b.Publish(42, domain.Comment{ID: 1})
}

func TestConcurrentSubscribePublishUnsubscribe(t *testing.T) {
	b := New()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch, unsubscribe := b.Subscribe(1)
			for j := 0; j < 100; j++ {
				b.Publish(1, domain.Comment{ID: 1})
			}
			unsubscribe()
			for range ch {
			}
		}()
	}
	wg.Wait()
}
