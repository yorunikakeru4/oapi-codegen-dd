package runtime

import (
	"encoding/json"
)

type Either[A, B any] struct {
	A A
	B B

	N int
}

func NewEitherFromA[A any, B any](a A) Either[A, B] {
	var b B
	return Either[A, B]{A: a, B: b, N: 1}
}

func NewEitherFromB[A any, B any](b B) Either[A, B] {
	var a A
	return Either[A, B]{A: a, B: b, N: 2}
}

func (t *Either[A, B]) IsA() bool {
	return t.N == 1
}

func (t *Either[A, B]) IsB() bool {
	return t.N == 2
}

func (t *Either[A, B]) Value() any {
	if t.IsA() {
		return t.A
	}
	if t.IsB() {
		return t.B
	}
	return nil
}

func (t *Either[A, B]) Unmarshal(data []byte) error {
	var a A
	if err := json.Unmarshal(data, &a); err == nil {
		t.A = a
		var b B
		t.B = b
		t.N = 1
		return nil
	}

	var b B
	if err := json.Unmarshal(data, &b); err == nil {
		t.B = b
		var a A
		t.A = a
		t.N = 2
		return nil
	}

	return ErrFailedToUnmarshalAsAOrB
}
