package epack

import (
	"testing"
)

// TestMarshal_NewEncoderOnlyOnCacheMiss 验证：
// Marshal 结构体时，epack tag 解析（newEncoder / encode.go:190 附近）
// 只在类型缓存未命中时执行一次；后续同类型 Marshal 走 cache，不再重建 units。
func TestMarshal_NewEncoderOnlyOnCacheMiss(t *testing.T) {
	type cacheProbe struct {
		A int32  `epack:"1"`
		B string `epack:"2"`
	}

	for i := 0; i < 10; i++ {
		if _, err := Marshal(cacheProbe{A: int32(i), B: "x"}); err != nil {
			t.Fatalf("Marshal err=%v", err)
		}
	}
}
