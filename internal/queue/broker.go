package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/ollymarsters/job-scraper/internal/dto"
	"github.com/ollymarsters/job-scraper/internal/sourcespec"
)

const (
	workExchange = "source.work"
	deadExchange = "source.dead.exchange"
	deadQueue    = "source.dead"
)

type Broker struct {
	url      string
	mu       sync.Mutex
	conn     *amqp.Connection
	pub      *amqp.Channel
	confirms <-chan amqp.Confirmation
	returns  <-chan amqp.Return
}

func (b *Broker) EnqueueJobs(ctx context.Context, jobs []dto.QueuedJob) error {
	for _, job := range jobs {
		task := Task{Version: 1, ID: uuid.NewString(), Source: job.Card.Source, Kind: DetailTask, URL: job.URL, Card: job.Card}
		if err := b.Publish(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func NewBroker(url string) (*Broker, error) {
	b := &Broker{url: url}
	if err := b.connect(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Broker) connect() error {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return fmt.Errorf("rabbitmq connect: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := declareTopology(ch); err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return err
	}
	b.conn, b.pub = conn, ch
	b.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	b.returns = ch.NotifyReturn(make(chan amqp.Return, 1))
	return nil
}

func declareTopology(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(workExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare work exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(deadExchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead exchange: %w", err)
	}
	if _, err := ch.QueueDeclare(deadQueue, true, false, false, false, amqp.Table{"x-queue-type": "quorum"}); err != nil {
		return fmt.Errorf("declare dead queue: %w", err)
	}
	if err := ch.QueueBind(deadQueue, deadQueue, deadExchange, false, nil); err != nil {
		return fmt.Errorf("bind dead queue: %w", err)
	}
	for _, src := range sourcespec.Sources() {
		name := "source." + src.Name
		args := amqp.Table{
			"x-queue-type":              "quorum",
			"x-max-length":              int32(10000),
			"x-delivery-limit":          int32(5),
			"x-delayed-retry-type":      "failed",
			"x-delayed-retry-min":       int32(10000),
			"x-delayed-retry-max":       int32(300000),
			"x-dead-letter-strategy":    "at-least-once",
			"x-overflow":                "reject-publish",
			"x-dead-letter-exchange":    deadExchange,
			"x-dead-letter-routing-key": deadQueue,
		}
		if _, err := ch.QueueDeclare(name, true, false, false, false, args); err != nil {
			return fmt.Errorf("declare %s: %w", name, err)
		}
		if err := ch.QueueBind(name, src.Name, workExchange, false, nil); err != nil {
			return fmt.Errorf("bind %s: %w", name, err)
		}
	}
	return nil
}

func (b *Broker) Publish(ctx context.Context, task Task) error {
	if err := task.Validate(); err != nil {
		return err
	}
	body, err := json.Marshal(task)
	if err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn == nil || b.conn.IsClosed() || b.pub == nil || b.pub.IsClosed() {
		if err := b.connect(); err != nil {
			return err
		}
	}
	priority := uint8(1)
	if task.Kind != DetailTask {
		priority = 8
	}
	msg := amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, MessageId: task.ID, Priority: priority, Body: body}
	if err := b.pub.PublishWithContext(ctx, workExchange, task.Source, true, false, msg); err != nil {
		return fmt.Errorf("publish %s: %w", task.ID, err)
	}
	returned := false
	for {
		select {
		case ret, ok := <-b.returns:
			if !ok {
				return errors.New("publisher closed before return check")
			}
			returned = true
			_ = ret
		case confirm, ok := <-b.confirms:
			if !ok {
				return errors.New("publisher closed before confirmation")
			}
			select {
			case <-b.returns:
				returned = true
			default:
			}
			if returned {
				return fmt.Errorf("unroutable task %s", task.ID)
			}
			if !confirm.Ack {
				return fmt.Errorf("broker rejected task %s", task.ID)
			}
			return nil
		case <-ctx.Done():
			_ = b.pub.Close()
			return fmt.Errorf("publish %s outcome unknown: %w", task.ID, ctx.Err())
		}
	}
}

func (b *Broker) Consume(ctx context.Context, handler func(context.Context, Task) error, terminal func(context.Context, Task) error) error {
	var wg sync.WaitGroup
	for _, source := range sourcespec.Sources() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			b.consumeSource(ctx, name, handler, terminal)
		}(source.Name)
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

func (b *Broker) consumeSource(ctx context.Context, source string, handler func(context.Context, Task) error, terminal func(context.Context, Task) error) {
	for ctx.Err() == nil {
		if err := b.consumeSession(ctx, source, handler, terminal); err != nil && ctx.Err() == nil {
			slog.Error("source consumer disconnected", slog.String("source", source), slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (b *Broker) consumeSession(ctx context.Context, source string, handler func(context.Context, Task) error, terminal func(context.Context, Task) error) error {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer func() { _ = ch.Close() }()
	if err := declareTopology(ch); err != nil {
		return err
	}
	if err := ch.Qos(1, 0, false); err != nil {
		return err
	}
	deliveries, err := ch.Consume("source."+source, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("delivery channel closed")
			}
			var task Task
			err := json.Unmarshal(delivery.Body, &task)
			task.Redelivered = delivery.Redelivered
			if err == nil {
				err = task.Validate()
			}
			if err == nil && task.Source != source {
				err = fmt.Errorf("task source %s delivered to %s", task.Source, source)
			}
			if err == nil {
				err = handler(ctx, task)
			}
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				slog.Error("source task failed", slog.String("task_id", task.ID), slog.String("source", source), slog.String("kind", string(task.Kind)), slog.String("target_id", task.TargetID), slog.String("run_id", task.RunID), slog.Any("err", err), slog.Any("x-delivery-count", delivery.Headers["x-delivery-count"]))
				if terminal != nil && deliveryCount(delivery.Headers["x-delivery-count"]) >= 5 {
					if terminalErr := terminal(ctx, task); terminalErr != nil {
						return fmt.Errorf("record terminal task failure for %s: %w", task.ID, terminalErr)
					}
				}
				if rejectErr := delivery.Reject(true); rejectErr != nil {
					return rejectErr
				}
				continue
			}
			if err := delivery.Ack(false); err != nil {
				return err
			}
		}
	}
}

func deliveryCount(value any) int64 {
	switch count := value.(type) {
	case int64:
		return count
	case int32:
		return int64(count)
	case int:
		return int64(count)
	default:
		return 0
	}
}

func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conn != nil {
		return b.conn.Close()
	}
	return nil
}
