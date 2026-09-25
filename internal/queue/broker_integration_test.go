package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/ollymarsters/job-scraper/internal/dto"
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
	broker, err := NewBroker(url)
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
		detail := Task{Version: 1, ID: uuid.NewString(), Source: "wis", Kind: DetailTask, URL: "https://workinstartups.com/job/1", Card: dto.Job{Source: "wis"}}
		listing := Task{Version: 1, ID: uuid.NewString(), Source: "wis", Kind: ListingPageTask, TargetID: uuid.NewString(), RunID: uuid.NewString()}
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
		if err := ch.QueueUnbind("source.wis", "wis", workExchange, nil); err != nil {
			t.Fatal(err)
		}
		if err := broker.Publish(ctx, detail); err == nil {
			t.Fatal("unroutable publish confirmed as success")
		}
		if err := ch.QueueBind("source.wis", "wis", workExchange, false, nil); err != nil {
			t.Fatal(err)
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
		deadline := time.Now().Add(8 * time.Second)
		attempts := 0
		var counts []int64
		for {
			dead, ok, err := ch.Get(deadQueue, false)
			if err != nil {
				t.Fatal(err)
			}
			if ok {
				if attempts != 3 || counts[2] != 2 || dead.Headers["x-death"] == nil {
					t.Fatalf("attempts=%d counts=%v headers=%v", attempts, counts, dead.Headers)
				}
				_ = dead.Ack(false)
				break
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
			if time.Now().After(deadline) {
				t.Fatalf("task never reached DLQ after %d attempts", attempts)
			}
			time.Sleep(25 * time.Millisecond)
		}
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
		deadline := time.Now().Add(5 * time.Second)
		for range 2 {
			for {
				delivery, ok, err := channel.Get("blocked.work", false)
				if err != nil {
					t.Fatal(err)
				}
				if ok {
					if err := delivery.Reject(true); err != nil {
						t.Fatal(err)
					}
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("blocked task not delivered")
				}
				time.Sleep(25 * time.Millisecond)
			}
		}
		for time.Now().Before(deadline) {
			if !publish("second") {
				return
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("source queue accepted work despite unavailable DLQ route")
	})

	t.Run("inspect and replay keeps failed replay", func(t *testing.T) {
		task := Task{Version: 1, ID: uuid.NewString(), Source: "wis", Kind: DetailTask, URL: "https://workinstartups.com/job/replay", Card: dto.Job{Source: "wis"}}
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
		if count, err := broker.DeadLetterCount(); err != nil || count != 1 {
			t.Fatalf("non-destructive inspect count=%d err=%v", count, err)
		}
		if err := ch.QueueUnbind("source.wis", "wis", workExchange, nil); err != nil {
			t.Fatal(err)
		}
		if err := broker.ReplayDead(ctx, task.ID); err == nil {
			t.Fatal("unroutable replay succeeded")
		}
		if count, err := broker.DeadLetterCount(); err != nil || count != 1 {
			t.Fatalf("failed replay count=%d err=%v", count, err)
		}
		if err := ch.QueueBind("source.wis", "wis", workExchange, false, nil); err != nil {
			t.Fatal(err)
		}
		if err := broker.ReplayDead(ctx, task.ID); err != nil {
			t.Fatal(err)
		}
		if count, err := broker.DeadLetterCount(); err != nil || count != 0 {
			t.Fatalf("successful replay count=%d err=%v", count, err)
		}
		replayed, ok, err := ch.Get("source.wis", false)
		if err != nil || !ok || replayed.MessageId == task.ID {
			t.Fatalf("replay delivery=%v ok=%v err=%v", replayed.MessageId, ok, err)
		}
		_ = replayed.Ack(false)
	})

	t.Run("one delivery per source with cross-source progress", func(t *testing.T) {
		for _, source := range []string{"wis", "wis", "linkedin"} {
			task := Task{Version: 1, ID: uuid.NewString(), Source: source, Kind: DetailTask, URL: "https://example.com/job/" + uuid.NewString(), Card: dto.Job{Source: source}}
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
			_ = broker.Consume(consumeCtx, func(ctx context.Context, task Task) error {
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

	t.Run("confirmed message survives restart", func(t *testing.T) {
		task := Task{Version: 1, ID: uuid.NewString(), Source: "linkedin", Kind: DetailTask, URL: "https://www.linkedin.com/jobs/view/1", Card: dto.Job{Source: "linkedin"}}
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
		deadline := time.Now().Add(30 * time.Second)
		var lastErr error
		for {
			fresh, err := amqp.Dial(restartedURL)
			lastErr = err
			if err == nil {
				channel, err := fresh.Channel()
				lastErr = err
				if err == nil {
					delivery, ok, err := channel.Get("source.linkedin", false)
					lastErr = err
					if err == nil && ok {
						if delivery.MessageId != task.ID || !delivery.Redelivered {
							t.Fatalf("restored message = %s redelivered=%v", delivery.MessageId, delivery.Redelivered)
						}
						_ = delivery.Ack(false)
						_ = channel.Close()
						_ = fresh.Close()
						break
					}
					_ = channel.Close()
				}
				_ = fresh.Close()
			}
			if time.Now().After(deadline) {
				t.Fatalf("confirmed message missing after restart: %v", lastErr)
			}
			time.Sleep(250 * time.Millisecond)
		}
	})
}
