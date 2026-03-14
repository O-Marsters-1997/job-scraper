package fp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNoneValueTaken = errors.New("none value taken")
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

func FromNillable[T any](v *T) Option[T] {
	if v == nil {
		return None[T]()
	}
	return Some[T](*v)
}

func PtrFromNillable[T any](v *T) Option[*T] {
	if v == nil {
		return None[*T]()
	}
	return Some[*T](v)
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

func (o Option[T]) UnwrapAsPtr() *T {
	if o.IsNone() {
		return nil
	}
	return &o[value]
}

func (o Option[T]) Take() (T, error) {
	if o.IsNone() {
		var defaultValue T
		return defaultValue, ErrNoneValueTaken
	}
	return o[value], nil
}

func (o Option[T]) TakeOr(fallbackValue T) T {
	if o.IsNone() {
		return fallbackValue
	}
	return o[value]
}

func (o Option[T]) TakeOrElse(fallbackFunc func() T) T {
	if o.IsNone() {
		return fallbackFunc()
	}
	return o[value]
}

func (o Option[T]) Or(fallbackOptionValue Option[T]) Option[T] {
	if o.IsNone() {
		return fallbackOptionValue
	}
	return o
}

func (o Option[T]) OrElse(fallbackOptionFunc func() Option[T]) Option[T] {
	if o.IsNone() {
		return fallbackOptionFunc()
	}
	return o
}

func (o Option[T]) Filter(predicate func(v T) bool) Option[T] {
	if o.IsNone() || !predicate(o[value]) {
		return None[T]()
	}
	return o
}

func (o Option[T]) IfSome(f func(v T)) {
	if o.IsNone() {
		return
	}
	f(o[value])
}

func (o Option[T]) IfSomeWithError(f func(v T) error) error {
	if o.IsNone() {
		return nil
	}
	return f(o[value])
}

func (o Option[T]) IfNone(f func()) {
	if o.IsSome() {
		return
	}
	f()
}

func (o Option[T]) IfNoneWithError(f func() error) error {
	if o.IsSome() {
		return nil
	}
	return f()
}

func (o Option[T]) String() string {
	if o.IsNone() {
		return "None[]"
	}

	v := o.Unwrap()
	if stringer, ok := interface{}(v).(fmt.Stringer); ok {
		return fmt.Sprintf("Some[%s]", stringer)
	}
	return fmt.Sprintf("Some[%v]", v)
}

func Map[T, U any](option Option[T], mapper func(v T) U) Option[U] {
	if option.IsNone() {
		return None[U]()
	}

	return Some(mapper(option[value]))
}

func MapOr[T, U any](option Option[T], fallbackValue U, mapper func(v T) U) U {
	if option.IsNone() {
		return fallbackValue
	}
	return mapper(option[value])
}

func MapWithError[T, U any](option Option[T], mapper func(v T) (U, error)) (Option[U], error) {
	if option.IsNone() {
		return None[U](), nil
	}

	u, err := mapper(option[value])
	if err != nil {
		return None[U](), err
	}
	return Some(u), nil
}

func MapOrWithError[T, U any](option Option[T], fallbackValue U, mapper func(v T) (U, error)) (U, error) {
	if option.IsNone() {
		return fallbackValue, nil
	}
	return mapper(option[value])
}

func FlatMap[T, U any](option Option[T], mapper func(v T) Option[U]) Option[U] {
	if option.IsNone() {
		return None[U]()
	}

	return mapper(option[value])
}

func FlatMapOr[T, U any](option Option[T], fallbackValue U, mapper func(v T) Option[U]) U {
	if option.IsNone() {
		return fallbackValue
	}

	return (mapper(option[value])).TakeOr(fallbackValue)
}

func FlatMapWithError[T, U any](option Option[T], mapper func(v T) (Option[U], error)) (Option[U], error) {
	if option.IsNone() {
		return None[U](), nil
	}

	mapped, err := mapper(option[value])
	if err != nil {
		return None[U](), err
	}
	return mapped, nil
}

func FlatMapOrWithError[T, U any](option Option[T], fallbackValue U, mapper func(v T) (Option[U], error)) (U, error) {
	if option.IsNone() {
		return fallbackValue, nil
	}

	maybe, err := mapper(option[value])
	if err != nil {
		var zeroValue U
		return zeroValue, err
	}

	return maybe.TakeOr(fallbackValue), nil
}

type Pair[T, U any] struct {
	Value1 T
	Value2 U
}

func Zip[T, U any](opt1 Option[T], opt2 Option[U]) Option[Pair[T, U]] {
	if opt1.IsSome() && opt2.IsSome() {
		return Some(Pair[T, U]{
			Value1: opt1[value],
			Value2: opt2[value],
		})
	}

	return None[Pair[T, U]]()
}

func ZipWith[T, U, V any](opt1 Option[T], opt2 Option[U], zipper func(opt1 T, opt2 U) V) Option[V] {
	if opt1.IsSome() && opt2.IsSome() {
		return Some(zipper(opt1[value], opt2[value]))
	}
	return None[V]()
}

func Unzip[T, U any](zipped Option[Pair[T, U]]) (Option[T], Option[U]) {
	if zipped.IsNone() {
		return None[T](), None[U]()
	}

	pair := zipped[value]
	return Some(pair.Value1), Some(pair.Value2)
}

func UnzipWith[T, U, V any](zipped Option[V], unzipper func(zipped V) (T, U)) (Option[T], Option[U]) {
	if zipped.IsNone() {
		return None[T](), None[U]()
	}

	v1, v2 := unzipper(zipped[value])
	return Some(v1), Some(v2)
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
