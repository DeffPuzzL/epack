package epack

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

// ---------- 测试用复杂模型（嵌套 + 多种切片/map/指针/time/数值） ----------

type benchL3 struct {
	Code int32  `epack:"1"`
	Note string `epack:"2"`
}

// 注意：避免在可序列化结构里使用「*嵌套结构体」字段；当前 epack 的 newEncoder 对指针嵌套用 vl.Field(i) 会传 Ptr 值，易触发 reflect.Field panic。
// 嵌套一律用值类型；指针仅保留在顶层业务里若你后续修了 encoder 再加回用例。
type benchL2 struct {
	Title  string  `epack:"1"`
	Items  []int64 `epack:"2"`
	Detail benchL3 `epack:"3"`
}

// BenchPayload 用于 epack/json 对比；字段需带连续 epack 序号。
type BenchPayload struct {
	ID        int64            `epack:"1"`
	Name      string           `epack:"2"`
	Tags      []string         `epack:"3"`
	Scores    []float64        `epack:"4"`
	IDs       []int32          `epack:"5"`
	Counters  map[string]int64 `epack:"6"`
	Primary   benchL2          `epack:"7"`
	Secondary benchL2          `epack:"8"`
	Created   time.Time        `epack:"9"`
	Updated   time.Time        `epack:"10"`
	Flags     []bool           `epack:"11"`
	Active    bool             `epack:"12"`
	Budget    float64          `epack:"13"`
}

func sampleBenchPayload() *BenchPayload {
	return &BenchPayload{
		ID:   982451653,
		Name: "payload-alpha-Ω",
		Tags: []string{"go", "epack", "bench", "json", "compare"},
		Scores: []float64{
			3.141592653589793, math.Sqrt2, math.E, -1.5e3,
		},
		IDs: []int32{7, 42, 256, 4096, 65535},
		Counters: map[string]int64{
			"req":     1_000_000,
			"err":     42,
			"latency": 9876543210,
		},
		Primary: benchL2{
			Title: "nested-primary",
			Items: []int64{1, 2, 3, 5, 8, 13},
			Detail: benchL3{
				Code: 404,
				Note: "not-found-mock",
			},
		},
		Secondary: benchL2{
			Title: "secondary-branch",
			Items: []int64{99, 100},
			Detail: benchL3{
				Code: -1,
				Note: "negative-code",
			},
		},
		Created: time.Date(2026, 4, 15, 12, 30, 45, 123456789, time.UTC),
		Updated: time.Date(2026, 4, 16, 8, 0, 0, 999999999, time.FixedZone("CST", 8*3600)),
		Flags:   []bool{true, false, true, true, false},
		Active:  true,
		Budget:  1.23e7,
	}
}

func equalBenchPayload(a, b *BenchPayload) bool {
	if a == nil || b == nil {
		return a == b
	}
	// time.Time 在序列化里只保留 UnixNano；反序列化后 Location 可能与原值不同，用 Equal 比瞬时时刻。
	return a.ID == b.ID &&
		a.Name == b.Name &&
		reflect.DeepEqual(a.Tags, b.Tags) &&
		reflect.DeepEqual(a.Scores, b.Scores) &&
		reflect.DeepEqual(a.IDs, b.IDs) &&
		reflect.DeepEqual(a.Counters, b.Counters) &&
		reflect.DeepEqual(a.Primary, b.Primary) &&
		reflect.DeepEqual(a.Secondary, b.Secondary) &&
		a.Created.Equal(b.Created) &&
		a.Updated.Equal(b.Updated) &&
		reflect.DeepEqual(a.Flags, b.Flags) &&
		a.Active == b.Active &&
		a.Budget == b.Budget
}

func TestBenchPayload_RoundTrip(t *testing.T) {
	src := sampleBenchPayload()
	data, err := Marshal(src)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty marshal output")
	}

	dst := new(BenchPayload)
	if err := Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !equalBenchPayload(src, dst) {
		t.Fatalf("round-trip mismatch\nsrc: %+v\ndst: %+v", src, dst)
	}
}

func TestBenchPayload_JSONRoundTrip(t *testing.T) {
	src := sampleBenchPayload()
	jb, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	dst := new(BenchPayload)
	if err := json.Unmarshal(jb, dst); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !equalBenchPayload(src, dst) {
		t.Fatalf("json round-trip mismatch")
	}
}

// ---------- 性能对比：序列化 ----------

var benchPayloadGlobal = sampleBenchPayload()
var epackBytesGlobal []byte
var jsonBytesGlobal []byte

func BenchmarkMarshal_Epack(b *testing.B) {
	var err error
	epackBytesGlobal, err = Marshal(benchPayloadGlobal)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(epackBytesGlobal)), "bytes/out")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Marshal(benchPayloadGlobal)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshal_JSON(b *testing.B) {
	var err error
	jsonBytesGlobal, err = json.Marshal(benchPayloadGlobal)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(jsonBytesGlobal)), "bytes/out")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(benchPayloadGlobal)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---------- 性能对比：反序列化 ----------

func BenchmarkUnmarshal_Epack(b *testing.B) {
	data, err := Marshal(benchPayloadGlobal)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data)), "bytes/in")
	dst := new(BenchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*dst = BenchPayload{}
		if err := Unmarshal(data, dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshal_JSON(b *testing.B) {
	data, err := json.Marshal(benchPayloadGlobal)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(len(data)), "bytes/in")
	dst := new(BenchPayload)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		*dst = BenchPayload{}
		if err := json.Unmarshal(data, dst); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------- 端到端（Marshal+Unmarshal）----------

func BenchmarkRoundTrip_Epack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data, err := Marshal(benchPayloadGlobal)
		if err != nil {
			b.Fatal(err)
		}
		dst := new(BenchPayload)
		if err := Unmarshal(data, dst); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRoundTrip_JSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(benchPayloadGlobal)
		if err != nil {
			b.Fatal(err)
		}
		dst := new(BenchPayload)
		if err := json.Unmarshal(data, dst); err != nil {
			b.Fatal(err)
		}
	}
}
