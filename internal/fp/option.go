package fp

import (
	"bytes"
	"encoding/json"
)

type Option[T any] []T

const (
	value = iota
)

func Some[T any](v T) Option[T] {
	return Option[T]{
		value: v,
	}
}

func None[T any]() Option[T] {
	return nil
}

func (o Option[T]) IsNone() bool {
	return o == nil
}

func (o Option[T]) IsSome() bool {
	return o != nil
}

func (o Option[T]) Unwrap() T {
	if o.IsNone() {
		var defaultValue T
		return defaultValue
	}
	return o[value]
}

func (o Option[T]) TakeOr(fallbackValue T) T {
	if o.IsNone() {
		return fallbackValue
	}
	return o[value]
}

var jsonNull = []byte("null")

func (o Option[T]) MarshalJSON() ([]byte, error) {
	if o.IsNone() {
		return jsonNull, nil
	}

	marshal, err := json.Marshal(o.Unwrap())
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if len(data) <= 0 || bytes.Equal(data, jsonNull) {
		*o = None[T]()
		return nil
	}

	var v T
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	*o = Some(v)

	return nil
}
