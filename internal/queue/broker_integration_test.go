package queue_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ollymarsters/job-scraper/internal/logger"
	"github.com/ollymarsters/job-scraper/internal/queue"
	"github.com/ollymarsters/job-scraper/internal/queue/queuetest"
	"github.com/ollymarsters/job-scraper/internal/telemetry"
)

const (
	workExchange = "source.work"
	deadExchange = "source.dead.exchange"
	deadQueue    = "source.dead"
)

func TestRabbitMQWorkQueue(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Docker")
	}
	ctx := context.Background()
	container, err := testcontainers.Run(ctx, "rabbitmq:4.3.6-management",
		testcontainers.WithEnv(map[string]string{"RABBITMQ_DEFAULT_USER": "jobs", "RABBITMQ_DEFAULT_PASS": "testpass"}),
		testcontainers.WithExposedPorts("5672/tcp", "15672/tcp"),
		testcontainers.WithWaitStrategy(wait.ForLog("Server startup complete")),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		t.Fatal(err)
	}
	managementPort, err := container.MappedPort(ctx, "15672/tcp")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("RABBITMQ_MANAGEMENT_URL", fmt.Sprintf("http://%s:%s", host, managementPort.Port()))
	url := fmt.Sprintf("amqp://jobs:testpass@%s:%s/", host, port.Port())
	broker, err := queue.NewBroker(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = broker.Close() }()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ch.Close() }()

	t.Run("priority and mandatory return", func(t *testing.T) {
		detail := queuetest.DetailTask("wis")
		listing := queuetest.ListingTask("wis")
		if err := broker.Publish(ctx, detail); err != nil {
			t.Fatal(err)
		}
		if err := broker.Publish(ctx, listing); err != nil {
			t.Fatal(err)
		}
		first, ok, err := ch.Get("source.wis", false)
		if err != nil || !ok {
			t.Fatalf("get first: %v %v", ok, err)
		}
		if first.MessageId != listing.ID {
			t.Fatalf("first message = %s, want listing", first.MessageId)
		}
		_ = first.Ack(false)
		second, ok, err := ch.Get("source.wis", false)
		if err != nil || !ok {
			t.Fatalf("get second: %v %v", ok, err)
		}
		_ = second.Ack(false)
		unbindSource(t, ch, "wis")
		if err := broker.Publish(ctx, detail); err == nil {
			t.Fatal("unroutable publish confirmed as success")
		}
	})

	t.Run("publish stamps timestamp", func(t *testing.T) {
		task := queuetest.DetailTask("wis")
		before := time.Now()
		if err := broker.Publish(ctx, task); err != nil {
			t.Fatal(err)
		}
		delivery, ok, err := ch.Get("source.wis", false)
		if err != nil || !ok {
			t.Fatalf("get: %v %v", ok, err)
		}
		_ = delivery.Ack(false)
		if delivery.Timestamp.IsZero() {
			t.Fatal("published message has zero Timestamp")
		}
		if delivery.Timestamp.Before(before.Add(-time.Second)) || delivery.Timestamp.After(time.Now().Add(time.Second)) {
			t.Fatalf("Timestamp = %v, want close to %v", delivery.Timestamp, before)
		}
	})

	t.Run("reject retries then dead letters", func(t *testing.T) {
		if err := ch.ExchangeDeclare("retry.work", "direct", true, false, false, false, nil); err != nil {
			t.Fatal(err)
		}
		args := amqp.Table{
			"x-queue-type": "quorum", "x-delivery-limit": int32(2),
			"x-delayed-retry-type": "failed", "x-delayed-retry-min": int32(100), "x-delayed-retry-max": int32(100),
			"x-dead-letter-strategy": "at-least-once", "x-overflow": "reject-publish",
			"x-dead-letter-exchange": deadExchange, "x-dead-letter-routing-key": deadQueue,
		}
		if _, err := ch.QueueDeclare("retry.work", true, false, false, false, args); err != nil {
			t.Fatal(err)
		}
		if err := ch.QueueBind("retry.work", "retry", "retry.work", false, nil); err != nil {
			t.Fatal(err)
		}
		if err := ch.PublishWithContext(ctx, "retry.work", "retry", true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, Body: []byte(`{"task_id":"retry-test"}`)}); err != nil {
			t.Fatal(err)
		}
		attempts := 0
		var counts []int64
		waitFor(t, 8*time.Second, func() error {
			dead, ok, err := ch.Get(deadQueue, false)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				if attempts != 3 || counts[2] != 2 || dead.Headers["x-death"] == nil {
					t.Fatalf("attempts=%d counts=%v headers=%v", attempts, counts, dead.Headers)
				}
				_ = dead.Ack(false)
				return nil
			}
			delivery, ok, err := ch.Get("retry.work", false)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				attempts++
				counts = append(counts, deliveryCount(delivery.Headers["x-delivery-count"]))
				if err := delivery.Reject(true); err != nil {
					t.Fatal(err)
				}
			}
			return fmt.Errorf("task not dead-lettered after %d attempts", attempts)
		})
	})

	t.Run("unavailable dead route retains and backpressures", func(t *testing.T) {
		channel, err := conn.Channel()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = channel.Close() }()
		if err := channel.ExchangeDeclare("blocked.work", "direct", true, false, false, false, nil); err != nil {
			t.Fatal(err)
		}
		if err := channel.ExchangeDeclare("blocked.dead", "direct", true, false, false, false, nil); err != nil {
			t.Fatal(err)
		}
		args := amqp.Table{
			"x-queue-type": "quorum", "x-max-length": int32(1), "x-delivery-limit": int32(1),
			"x-delayed-retry-type": "failed", "x-delayed-retry-min": int32(100), "x-delayed-retry-max": int32(100),
			"x-dead-letter-strategy": "at-least-once", "x-overflow": "reject-publish",
			"x-dead-letter-exchange": "blocked.dead", "x-dead-letter-routing-key": "missing",
		}
		if _, err := channel.QueueDeclare("blocked.work", true, false, false, false, args); err != nil {
			t.Fatal(err)
		}
		if err := channel.QueueBind("blocked.work", "blocked", "blocked.work", false, nil); err != nil {
			t.Fatal(err)
		}
		if err := channel.Confirm(false); err != nil {
			t.Fatal(err)
		}
		confirms := channel.NotifyPublish(make(chan amqp.Confirmation, 1))
		publish := func(body string) bool {
			if err := channel.PublishWithContext(ctx, "blocked.work", "blocked", true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, Body: []byte(body)}); err != nil {
				t.Fatal(err)
			}
			return (<-confirms).Ack
		}
		if !publish("first") {
			t.Fatal("first publish rejected")
		}
		for range 2 {
			waitFor(t, 5*time.Second, func() error {
				delivery, ok, err := channel.Get("blocked.work", false)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					return errors.New("blocked task not delivered")
				}
				return delivery.Reject(true)
			})
		}
		waitFor(t, 5*time.Second, func() error {
			if publish("second") {
				return errors.New("source queue accepted work despite unavailable DLQ route")
			}
			return nil
		})
	})

	t.Run("inspect and replay keeps failed replay", func(t *testing.T) {
		task := queuetest.DetailTask("wis")
		body, err := json.Marshal(task)
		if err != nil {
			t.Fatal(err)
		}
		if err := ch.PublishWithContext(ctx, deadExchange, deadQueue, true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, Body: body}); err != nil {
			t.Fatal(err)
		}
		letters, err := broker.DeadLetters(ctx, 10)
		if err != nil || len(letters) != 1 || letters[0].Task.ID != task.ID {
			t.Fatalf("letters=%v err=%v", letters, err)
		}
		waitForDeadLetterCount(t, broker, 1)
		unbindSource(t, ch, "wis")
		if err := broker.ReplayDead(ctx, task.ID); err == nil {
			t.Fatal("unroutable replay succeeded")
		}
		waitForDeadLetterCount(t, broker, 1)
		if err := ch.QueueBind("source.wis", "wis", workExchange, false, nil); err != nil {
			t.Fatal(err)
		}
		if err := broker.ReplayDead(ctx, task.ID); err != nil {
			t.Fatal(err)
		}
		waitForDeadLetterCount(t, broker, 0)
		replayed, ok, err := ch.Get("source.wis", false)
		if err != nil || !ok || replayed.MessageId == task.ID {
			t.Fatalf("replay delivery=%v ok=%v err=%v", replayed.MessageId, ok, err)
		}
		_ = replayed.Ack(false)
	})

	t.Run("one delivery per source with cross-source progress", func(t *testing.T) {
		for _, source := range []string{"wis", "wis", "linkedin"} {
			task := queuetest.DetailTask(source)
			if err := broker.Publish(ctx, task); err != nil {
				t.Fatal(err)
			}
		}
		consumeCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		entered := make(chan string, 3)
		release := make(chan struct{})
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = broker.Consume(consumeCtx, func(ctx context.Context, task queue.Task) error {
				entered <- task.Source
				if task.Source == "wis" {
					select {
					case <-release:
					case <-ctx.Done():
					}
				}
				return nil
			}, nil)
		}()
		seen := map[string]bool{}
		for len(seen) < 2 {
			select {
			case got := <-entered:
				if seen[got] {
					t.Fatalf("source %s delivered a second task before ack", got)
				}
				seen[got] = true
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for both sources")
			}
		}
		select {
		case source := <-entered:
			t.Fatalf("second wis delivered before ack: %s", source)
		case <-time.After(100 * time.Millisecond):
		}
		close(release)
		select {
		case got := <-entered:
			if got != "wis" {
				t.Fatalf("next source=%s", got)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("second wis task did not run")
		}
		cancel()
		<-done
	})

	t.Run("task.done logs one line per delivery", func(t *testing.T) {
		runOne := func(t *testing.T, handler func(context.Context, queue.Task) error) map[string]any {
			t.Helper()
			buf := captureTaskDoneLogs(t)
			consumeCtx, cancel := context.WithCancel(ctx)
			entered := make(chan struct{})
			done := make(chan struct{})
			go func() {
				defer close(done)
				_ = broker.Consume(consumeCtx, func(ctx context.Context, got queue.Task) error {
					defer close(entered)
					return handler(ctx, got)
				}, nil)
			}()
			<-entered
			cancel()
			<-done
			lines := taskDoneLines(t, buf)
			if len(lines) != 1 {
				t.Fatalf("task.done lines = %d, want 1: %v", len(lines), lines)
			}
			return lines[0]
		}

		for _, tt := range []struct {
			name        string
			handlerErr  error
			rawPublish  bool
			wantOutcome string
			wantWaitMs  bool
		}{
			{"ok outcome carries wait_ms", nil, false, "ok", true},
			{"error outcome", errors.New("handler boom"), false, "error", true},
			{"zero Timestamp omits wait_ms", nil, true, "ok", false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				task := queuetest.ListingTask("indeed")
				if tt.rawPublish {
					body, err := json.Marshal(task)
					if err != nil {
						t.Fatal(err)
					}
					if err := ch.PublishWithContext(ctx, workExchange, "indeed", true, false, amqp.Publishing{DeliveryMode: amqp.Persistent, MessageId: task.ID, Body: body}); err != nil {
						t.Fatal(err)
					}
				} else if err := broker.Publish(ctx, task); err != nil {
					t.Fatal(err)
				}
				line := runOne(t, func(context.Context, queue.Task) error { return tt.handlerErr })

				want := map[string]any{"task_id": task.ID, "run_id": task.RunID, "source": task.Source, "kind": string(task.Kind), "outcome": tt.wantOutcome}
				got := map[string]any{}
				for key := range want {
					got[key] = line[key]
				}
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("task.done fields (-want +got):\n%s", diff)
				}
				if duration, ok := line["duration_ms"].(float64); !ok || duration < 0 {
					t.Errorf("duration_ms = %v, want >= 0", line["duration_ms"])
				}
				waitMs, hasWait := line["wait_ms"].(float64)
				if hasWait != tt.wantWaitMs || hasWait && waitMs < 0 {
					t.Errorf("wait_ms = %v, want present=%t and >= 0", line["wait_ms"], tt.wantWaitMs)
				}
			})
		}
	})

	t.Run("confirmed message survives restart", func(t *testing.T) {
		task := queuetest.DetailTask("linkedin")
		if err := broker.Publish(ctx, task); err != nil {
			t.Fatal(err)
		}
		claimed, ok, err := ch.Get("source.linkedin", false)
		if err != nil || !ok || claimed.MessageId != task.ID {
			t.Fatalf("claim before restart: %v %v", ok, err)
		}
		if err := container.Stop(ctx, nil); err != nil {
			t.Fatal(err)
		}
		if err := container.Start(ctx); err != nil {
			t.Fatal(err)
		}
		restartedPort, err := container.MappedPort(ctx, "5672/tcp")
		if err != nil {
			t.Fatal(err)
		}
		restartedURL := fmt.Sprintf("amqp://jobs:testpass@%s:%s/", host, restartedPort.Port())
		waitFor(t, 30*time.Second, func() error {
			fresh, err := amqp.Dial(restartedURL)
			if err != nil {
				return err
			}
			defer func() { _ = fresh.Close() }()
			channel, err := fresh.Channel()
			if err != nil {
				return err
			}
			defer func() { _ = channel.Close() }()
			delivery, ok, err := channel.Get("source.linkedin", false)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("confirmed message missing after restart")
			}
			if delivery.MessageId != task.ID || !delivery.Redelivered {
				t.Fatalf("restored message = %s redelivered=%v", delivery.MessageId, delivery.Redelivered)
			}
			_ = delivery.Ack(false)
			return nil
		})
	})
}

func deliveryCount(value any) int64 {
	count, _ := value.(int64)
	return count
}

func waitFor(t *testing.T, timeout time.Duration, check func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		err := check()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s: %v", timeout, err)
		}
		<-ticker.C
	}
}

func waitForDeadLetterCount(t *testing.T, broker *queue.Broker, want int) {
	t.Helper()
	waitFor(t, 2*time.Second, func() error {
		count, err := broker.DeadLetterCount()
		if err != nil {
			t.Fatal(err)
		}
		if count != want {
			return fmt.Errorf("dead letter count = %d, want %d", count, want)
		}
		return nil
	})
}

func unbindSource(t *testing.T, ch *amqp.Channel, source string) {
	t.Helper()
	if err := ch.QueueUnbind("source."+source, source, workExchange, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ch.QueueBind("source."+source, source, workExchange, false, nil); err != nil {
			t.Errorf("rebind source.%s: %v", source, err)
		}
	})
}

func captureTaskDoneLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	lg, err := logger.New(buf, "json", "debug")
	if err != nil {
		t.Fatalf("logger.New() = %v", err)
	}
	prev := slog.Default()
	slog.SetDefault(lg)
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

func taskDoneLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		var line map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Fatalf("unmarshal log line: %v", err)
		}
		if line["event"] == telemetry.EventTaskDone {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan log buffer: %v", err)
	}
	return lines
}
