package queue

import (
	"go.opentelemetry.io/otel/propagation"

	amqp "github.com/rabbitmq/amqp091-go"
)

type headerCarrier amqp.Table

var _ propagation.TextMapCarrier = headerCarrier(nil)

func (c headerCarrier) Get(key string) string {
	s, _ := c[key].(string)
	return s
}

func (c headerCarrier) Set(key, value string) { c[key] = value }

func (c headerCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
