package epack

import (
	"errors"
	"reflect"
	"testing"
)

func TestDemo_FirstUnmarshalAlsoChecksIsOver(t *testing.T) {
	type isOverDemo struct {
		A int32  `epack:"1"`
		B string `epack:"2"`
	}
	key := reflect.TypeOf(isOverDemo{}).String()
	conf.cache.Delete(key)

	clean, err := Marshal(isOverDemo{A: 42, B: "ok"})
	if err != nil {
		t.Fatalf("Marshal err=%v", err)
	}
	dirty := append(append([]byte{}, clean...), 0xDE, 0xAD)

	// Marshal 也会写 cache；清掉才能真正打到 newUnmarshal
	conf.cache.Delete(key)

	var first isOverDemo
	err1 := Unmarshal(dirty, &first)
	t.Logf("第1次 Unmarshal(cache miss): err=%v, value=%+v", err1, first)
	if !errors.Is(err1, errBufferNotFinished) {
		t.Fatalf("首次(miss) 应因 isOver 失败，实际 err=%v", err1)
	}

	var second isOverDemo
	err2 := Unmarshal(dirty, &second)
	t.Logf("第2次 Unmarshal(cache hit):  err=%v, value=%+v", err2, second)
	if !errors.Is(err2, errBufferNotFinished) {
		t.Fatalf("二次(hit) 应因 isOver 失败，实际 err=%v", err2)
	}
}
