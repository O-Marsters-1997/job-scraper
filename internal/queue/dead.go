package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DeadLetter struct {
	Task  Task `json:"task"`
	Death any  `json:"x_death,omitempty"`
}

func (b *Broker) DeadLetters(ctx context.Context, count int) ([]DeadLetter, error) {
	if count < 1 || count > 100 {
		return nil, errors.New("count must be 1..100")
	}
	endpoint := os.Getenv("RABBITMQ_MANAGEMENT_URL")
	if endpoint == "" {
		endpoint = "http://localhost:15672"
	}
	body, _ := json.Marshal(map[string]any{"count": count, "ackmode": "ack_requeue_true", "encoding": "auto"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/queues/%2F/"+deadQueue+"/get", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	brokerURL, err := url.Parse(b.url)
	if err != nil {
		return nil, err
	}
	if brokerURL.User != nil {
		password, _ := brokerURL.User.Password()
		req.SetBasicAuth(brokerURL.User.Username(), password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("management API %s: %s", resp.Status, message)
	}
	var rows []struct {
		Payload    string `json:"payload"`
		Properties struct {
			Headers map[string]any `json:"headers"`
		} `json:"properties"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	letters := make([]DeadLetter, 0, len(rows))
	for _, row := range rows {
		var task Task
		if err := json.Unmarshal([]byte(row.Payload), &task); err != nil {
			return nil, err
		}
		letters = append(letters, DeadLetter{Task: task, Death: row.Properties.Headers["x-death"]})
	}
	return letters, nil
}

func (b *Broker) DeadLetterCount() (int, error) {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return 0, err
	}
	defer func() { _ = ch.Close() }()
	q, err := ch.QueueDeclarePassive(deadQueue, true, false, false, false, nil)
	if err != nil {
		return 0, err
	}
	return q.Messages, nil
}

func (b *Broker) ReplayDead(ctx context.Context, id string) error {
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
	state, err := ch.QueueDeclarePassive(deadQueue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	var held []amqp.Delivery
	defer func() {
		for _, delivery := range held {
			_ = delivery.Reject(true)
		}
	}()
	for range state.Messages {
		delivery, ok, err := ch.Get(deadQueue, false)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		var task Task
		if err := json.Unmarshal(delivery.Body, &task); err != nil {
			held = append(held, delivery)
			continue
		}
		if task.ID != id {
			held = append(held, delivery)
			continue
		}
		task.ID = uuid.NewString()
		if err := b.Publish(ctx, task); err != nil {
			held = append(held, delivery)
			return err
		}
		return delivery.Ack(false)
	}
	return fmt.Errorf("dead letter %s not found", id)
}
