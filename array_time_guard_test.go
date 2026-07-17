package epack

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestArrayDecoderLengthMismatch(t *testing.T) {
	var dst [2]int32
	// wire 声明 3 个元素，目标数组只有 2 —— 以前会 Index panic
	h := encodeHead(uint64(reflect.Array), 3)
	err := arrayDecoder(nil, deBuffer(h), reflect.ValueOf(&dst).Elem())
	if !errors.Is(err, errBadArrayLen) {
		t.Fatalf("want errBadArrayLen, got %v", err)
	}

	// wire 声明 1 个，目标 2 —— 定长要求相等
	h1 := encodeHead(uint64(reflect.Array), 1)
	err = arrayDecoder(nil, deBuffer(h1), reflect.ValueOf(&dst).Elem())
	if !errors.Is(err, errBadArrayLen) {
		t.Fatalf("want errBadArrayLen for short wire len, got %v", err)
	}
}

func TestTimeDecoderBadSize(t *testing.T) {
	var tm time.Time

	// size=0：以前可能 panic
	h0 := encodeHead(uint64(reflect.Int64), 0)
	if err := timeDecoder(nil, deBuffer(h0), reflect.ValueOf(&tm).Elem()); !errors.Is(err, errBadTimeSize) {
		t.Fatalf("size=0 want errBadTimeSize, got %v", err)
	}

	// size=4：非法时间载荷
	h4 := encodeHead(uint64(reflect.Int64), 4)
	buf := append(h4, 0, 0, 0, 1)
	if err := timeDecoder(nil, deBuffer(buf), reflect.ValueOf(&tm).Elem()); !errors.Is(err, errBadTimeSize) {
		t.Fatalf("size=4 want errBadTimeSize, got %v", err)
	}
}
