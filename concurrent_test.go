package epack

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type concLeaf struct {
	ID   int32  `epack:"1"`
	Name string `epack:"2"`
}

type concRoot struct {
	A int32             `epack:"1"`
	B string            `epack:"2"`
	C []int64           `epack:"3"`
	D map[string]int32  `epack:"4"`
	E *concLeaf         `epack:"5"`
	F []concLeaf        `epack:"6"`
	G interface{}       `epack:"7"`
	T time.Time         `epack:"8"`
}

func sampleConcRoot(i int) concRoot {
	leaf := &concLeaf{ID: int32(i), Name: fmt.Sprintf("n-%d", i)}
	return concRoot{
		A: int32(i),
		B: fmt.Sprintf("s-%d", i),
		C: []int64{int64(i), int64(i + 1), int64(i + 2)},
		D: map[string]int32{"k": int32(i)},
		E: leaf,
		F: []concLeaf{{ID: int32(i), Name: "a"}, {ID: int32(i + 1), Name: "b"}},
		G: int32(i * 10),
		T: time.Unix(0, int64(i)*1000).UTC(),
	}
}

func TestConcurrent_MarshalUnmarshalSameType(t *testing.T) {
	conf.cache.Delete(reflect.TypeOf(concRoot{}).String())
	const goroutines = 32
	const rounds = 200

	var wg sync.WaitGroup
	var fails atomic.Int64
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				src := sampleConcRoot(gid*1000 + r)
				data, err := Marshal(src)
				if err != nil {
					fails.Add(1)
					t.Errorf("Marshal: %v", err)
					return
				}
				var got concRoot
				if err := Unmarshal(data, &got); err != nil {
					fails.Add(1)
					t.Errorf("Unmarshal: %v", err)
					return
				}
				if got.A != src.A || got.B != src.B || got.E == nil || got.E.ID != src.E.ID {
					fails.Add(1)
					t.Errorf("mismatch gid=%d r=%d got=%+v src=%+v", gid, r, got, src)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() > 0 {
		t.Fatalf("failures=%d", fails.Load())
	}
}

func TestConcurrent_CacheMissStorm(t *testing.T) {
	// 大量 goroutine 同时首次 Marshal 同一新类型，冲击 newMarshal + cache.Store
	type storm struct {
		X int32  `epack:"1"`
		Y string `epack:"2"`
		Z []byte `epack:"3"`
	}
	key := reflect.TypeOf(storm{}).String()
	conf.cache.Delete(key)

	const n = 64
	var wg sync.WaitGroup
	var fails atomic.Int64
	start := make(chan struct{})
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			src := storm{X: int32(i), Y: fmt.Sprintf("y%d", i), Z: []byte{byte(i), 1, 2}}
			data, err := Marshal(src)
			if err != nil {
				fails.Add(1)
				t.Errorf("Marshal: %v", err)
				return
			}
			var got storm
			if err := Unmarshal(data, &got); err != nil || got.X != src.X || got.Y != src.Y || !bytes.Equal(got.Z, src.Z) {
				fails.Add(1)
				t.Errorf("roundtrip fail i=%d err=%v got=%+v", i, err, got)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if fails.Load() > 0 {
		t.Fatalf("failures=%d", fails.Load())
	}
	if _, ok := conf.cache.Load(key); !ok {
		t.Fatal("cache missing after storm")
	}
}

func TestConcurrent_MixedTopLevelTypes(t *testing.T) {
	const goroutines = 16
	const rounds = 100
	var wg sync.WaitGroup
	var fails atomic.Int64
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				switch r % 5 {
				case 0:
					src := []int32{int32(gid), int32(r), 7}
					data, err := Marshal(src)
					if err != nil {
						fails.Add(1)
						return
					}
					var got []int32
					if err := Unmarshal(data, &got); err != nil || !reflect.DeepEqual(src, got) {
						fails.Add(1)
						t.Errorf("slice: err=%v got=%v", err, got)
					}
				case 1:
					src := map[string]int64{"a": int64(gid), "b": int64(r)}
					data, err := Marshal(src)
					if err != nil {
						fails.Add(1)
						return
					}
					var got map[string]int64
					if err := Unmarshal(data, &got); err != nil || got["a"] != src["a"] || got["b"] != src["b"] {
						fails.Add(1)
						t.Errorf("map: err=%v got=%v", err, got)
					}
				case 2:
					src := fmt.Sprintf("str-%d-%d", gid, r)
					data, err := Marshal(src)
					if err != nil {
						fails.Add(1)
						return
					}
					var got string
					if err := Unmarshal(data, &got); err != nil || got != src {
						fails.Add(1)
					}
				case 3:
					src := int64(gid*1000 + r)
					data, err := Marshal(src)
					if err != nil {
						fails.Add(1)
						return
					}
					var got int64
					if err := Unmarshal(data, &got); err != nil || got != src {
						fails.Add(1)
					}
				default:
					src := sampleConcRoot(gid + r)
					data, err := Marshal(&src)
					if err != nil {
						fails.Add(1)
						return
					}
					var got concRoot
					if err := Unmarshal(data, &got); err != nil || got.A != src.A {
						fails.Add(1)
						t.Errorf("struct: err=%v", err)
					}
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() > 0 {
		t.Fatalf("failures=%d", fails.Load())
	}
}

func TestConcurrent_LoadTemplateRace(t *testing.T) {
	// LoadTemplate 先 Store 空 ePack 再填 units；并发 Marshal 可能读到 units==nil。
	type ltRace struct {
		A int32  `epack:"1"`
		B string `epack:"2"`
	}
	key := reflect.TypeOf(ltRace{}).String()
	conf.cache.Delete(key)

	const n = 32
	var wg sync.WaitGroup
	var marshalFails atomic.Int64
	var panics atomic.Int64
	start := make(chan struct{})
	wg.Add(n * 2)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			_ = LoadTemplate(ltRace{})
		}()
		go func(i int) {
			defer wg.Done()
			defer func() {
				if rec := recover(); rec != nil {
					panics.Add(1)
					t.Errorf("panic: %v", rec)
				}
			}()
			<-start
			for k := 0; k < 50; k++ {
				_, err := Marshal(ltRace{A: int32(i), B: "x"})
				if err != nil {
					marshalFails.Add(1)
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	t.Logf("LoadTemplate race: marshalFails=%d panics=%d", marshalFails.Load(), panics.Load())
	if panics.Load() > 0 {
		t.Fatalf("concurrent LoadTemplate caused panic")
	}
}

func TestConcurrent_BufferPoolStress(t *testing.T) {
	// 压 sync.Pool：频繁 Marshal 大/小报文交错
	const goroutines = 24
	const rounds = 150
	var wg sync.WaitGroup
	var fails atomic.Int64
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				var src interface{}
				if r%2 == 0 {
					src = sampleConcRoot(gid*r + 1)
				} else {
					s := make([]byte, 2048+r%128)
					for i := range s {
						s[i] = byte(i + gid)
					}
					src = concRoot{A: int32(gid), B: string(s[:64]), C: []int64{1, 2, 3}}
				}
				data, err := Marshal(src)
				if err != nil || len(data) == 0 {
					fails.Add(1)
					return
				}
				// 立刻再编一次，迫使 pool Get/Put 交错
				data2, err := Marshal(src)
				if err != nil || !bytes.Equal(data, data2) {
					fails.Add(1)
					t.Errorf("pool stress mismatch gid=%d r=%d err=%v", gid, r, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if fails.Load() > 0 {
		t.Fatalf("failures=%d", fails.Load())
	}
}
