package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/messageid"
)

type testBus struct {
	mu        sync.Mutex
	fail      map[string]error
	published []eventbus.Message
}

func (b *testBus) Publish(_ context.Context, message eventbus.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.fail[message.Topic]; err != nil {
		return err
	}
	b.published = append(b.published, message)
	return nil
}
func (*testBus) Consume(context.Context, eventbus.Subscription, eventbus.Handler) error { return nil }

type testSource struct {
	mu      sync.Mutex
	name    string
	records []Record
}

func (s *testSource) Name() string { return s.name }
func (s *testSource) Claim(_ context.Context, claim Claim) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Record, 0, claim.Limit)
	for index := range s.records {
		record := &s.records[index]
		if record.PublishedAt != nil || record.AvailableAt.After(claim.Now) || (record.LeaseUntil != nil && record.LeaseUntil.After(claim.Now)) {
			continue
		}
		until := claim.Now.Add(claim.LeaseDuration)
		record.LeaseOwner, record.LeaseUntil = claim.Owner, &until
		result = append(result, *record)
		if len(result) == claim.Limit {
			break
		}
	}
	return result, nil
}
func (s *testSource) MarkPublished(_ context.Context, id messageid.ID, owner string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.records {
		if s.records[index].MessageID == id && s.records[index].LeaseOwner == owner {
			s.records[index].PublishedAt = &at
			s.records[index].LeaseOwner = ""
			s.records[index].LeaseUntil = nil
			return nil
		}
	}
	return ErrLeaseLost
}
func (s *testSource) MarkFailed(_ context.Context, id messageid.ID, owner, failure string, availableAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.records {
		if s.records[index].MessageID == id && s.records[index].LeaseOwner == owner {
			s.records[index].AttemptCount++
			s.records[index].LastError = failure
			s.records[index].AvailableAt = availableAt
			s.records[index].LeaseOwner = ""
			s.records[index].LeaseUntil = nil
			return nil
		}
	}
	return ErrLeaseLost
}
func (s *testSource) CleanupPublished(_ context.Context, before time.Time, limit int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.records[:0]
	var deleted int64
	for _, record := range s.records {
		if deleted < int64(limit) && record.PublishedAt != nil && !record.PublishedAt.After(before) {
			deleted++
			continue
		}
		kept = append(kept, record)
	}
	s.records = kept
	return deleted, nil
}

func TestPublisherSuccessFailureRetryAndContinuation(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	source := &testSource{name: "core:test", records: []Record{
		newTestRecord(t, "failing", now), newTestRecord(t, "working", now),
	}}
	bus := &testBus{fail: map[string]error{"failing": errors.New("broker unavailable")}}
	publisher, err := NewPublisher(bus, []Source{source}, slog.New(slog.NewTextHandler(io.Discard, nil)), PublisherConfig{InitialRetryDelay: time.Second, MaximumRetryDelay: 4 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	publisher.now = func() time.Time { return now }
	publisher.process(context.Background())
	if source.records[0].AttemptCount != 1 || source.records[0].LastError == "" || !source.records[0].AvailableAt.Equal(now.Add(time.Second)) {
		t.Fatalf("failed record = %#v", source.records[0])
	}
	if source.records[1].PublishedAt == nil || len(bus.published) != 1 || bus.published[0].Topic != "working" {
		t.Fatalf("successful continuation records=%#v published=%#v", source.records, bus.published)
	}
	delete(bus.fail, "failing")
	publisher.process(context.Background())
	if source.records[0].PublishedAt != nil {
		t.Fatal("retry delay was ignored")
	}
	now = now.Add(time.Second)
	publisher.process(context.Background())
	if source.records[0].PublishedAt == nil {
		t.Fatal("retryable record was not published")
	}
}

func TestPublisherCleanupAndCancellation(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-8 * 24 * time.Hour)
	recent := now.Add(-time.Hour)
	unpublished := newTestRecord(t, "unpublished", now)
	oldPublished := newTestRecord(t, "old", now)
	oldPublished.PublishedAt = &old
	recentPublished := newTestRecord(t, "recent", now)
	recentPublished.PublishedAt = &recent
	source := &testSource{name: "core:test", records: []Record{oldPublished, recentPublished, unpublished}}
	publisher, err := NewPublisher(&testBus{fail: map[string]error{}}, []Source{source}, slog.New(slog.NewTextHandler(io.Discard, nil)), PublisherConfig{PollInterval: time.Millisecond, CleanupInterval: time.Millisecond, PublishedRetention: 7 * 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	publisher.now = func() time.Time { return now }
	publisher.cleanup(context.Background(), now)
	if len(source.records) != 2 {
		t.Fatalf("cleanup kept records = %#v", source.records)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- publisher.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("publisher did not stop after cancellation")
	}
}

func newTestRecord(t *testing.T, topic string, now time.Time) Record {
	t.Helper()
	id, err := messageid.New()
	if err != nil {
		t.Fatal(err)
	}
	return Record{MessageID: id, Topic: topic, Body: []byte(topic), AvailableAt: now, CreatedAt: now}
}
