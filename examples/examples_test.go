package epack_test

import (
	"fmt"
	"time"

	"github.com/DeffPuzzL/epack"
)

// 基础类型示例：基本类型的序列化与反序列化
func ExampleMarshal_basicTypes() {
	// 序列化 bool
	b, _ := epack.Marshal(true)
	var out bool
	epack.Unmarshal(b, &out)
	fmt.Println(out)

	// 序列化 string
	b, _ = epack.Marshal("hello epack")
	var sout string
	epack.Unmarshal(b, &sout)
	fmt.Println(sout)

	// 序列化 int
	b, _ = epack.Marshal(int(42))
	var iout int
	epack.Unmarshal(b, &iout)
	fmt.Println(iout)

	// 序列化 float64
	b, _ = epack.Marshal(3.14)
	var fout float64
	epack.Unmarshal(b, &fout)
	fmt.Println(fout)

	// Output:
	// true
	// hello epack
	// 42
	// 3.14
}

// 结构体示例：使用 epack tag 定义字段顺序
type ExamplePerson struct {
	Name    string   `epack:"1"`
	Age     int      `epack:"2"`
	Height  float64  `epack:"3"`
	Hobbies []string `epack:"4"`
}

func ExampleMarshal_struct() {
	p := ExamplePerson{
		Name:    "Alice",
		Age:     30,
		Height:  1.68,
		Hobbies: []string{"reading", "hiking", "coding"},
	}

	// 序列化
	data, err := epack.Marshal(p)
	if err != nil {
		fmt.Println("marshal error:", err)
		return
	}
	fmt.Printf("encoded %d bytes\n", len(data))

	// 反序列化
	var p2 ExamplePerson
	if err := epack.Unmarshal(data, &p2); err != nil {
		fmt.Println("unmarshal error:", err)
		return
	}
	fmt.Printf("Name=%s Age=%d Height=%.2f Hobbies=%v\n",
		p2.Name, p2.Age, p2.Height, p2.Hobbies)

	// Output:
	// encoded 54 bytes
	// Name=Alice Age=30 Height=1.68 Hobbies=[reading hiking coding]
}

// 嵌套结构体示例
type ExampleAddress struct {
	City    string `epack:"1"`
	ZipCode string `epack:"2"`
}

type ExampleEmployee struct {
	Name    string         `epack:"1"`
	Address ExampleAddress `epack:"2"`
	Salary  float64        `epack:"3"`
}

func ExampleMarshal_nestedStruct() {
	e := ExampleEmployee{
		Name:    "Bob",
		Address: ExampleAddress{City: "Beijing", ZipCode: "100000"},
		Salary:  8000.50,
	}

	data, _ := epack.Marshal(e)

	var e2 ExampleEmployee
	epack.Unmarshal(data, &e2)

	fmt.Printf("Name=%s City=%s Zip=%s Salary=%.2f\n",
		e2.Name, e2.Address.City, e2.Address.ZipCode, e2.Salary)

	// Output:
	// Name=Bob City=Beijing Zip=100000 Salary=8000.50
}

// 切片（数组）示例
type ExampleTeam struct {
	Name    string   `epack:"1"`
	Members []string `epack:"2"`
	Scores  []int    `epack:"3"`
}

func ExampleMarshal_slices() {
	t := ExampleTeam{
		Name:    "Go Team",
		Members: []string{"Alice", "Bob", "Charlie"},
		Scores:  []int{95, 87, 92},
	}

	data, _ := epack.Marshal(t)

	var t2 ExampleTeam
	epack.Unmarshal(data, &t2)

	fmt.Printf("Name=%s Members=%v Scores=%v\n", t2.Name, t2.Members, t2.Scores)

	// Output:
	// Name=Go Team Members=[Alice Bob Charlie] Scores=[95 87 92]
}

// 时间类型示例
type ExampleEvent struct {
	Title     string    `epack:"1"`
	CreatedAt time.Time `epack:"2"`
}

func ExampleMarshal_time() {
	e := ExampleEvent{
		Title:     "Release v1.0",
		CreatedAt: time.Date(2025, 7, 17, 10, 0, 0, 0, time.UTC),
	}

	data, _ := epack.Marshal(e)

	var e2 ExampleEvent
	epack.Unmarshal(data, &e2)

	fmt.Printf("Title=%s CreatedAt=%s\n", e2.Title, e2.CreatedAt.UTC().Format(time.RFC3339))

	// Output:
	// Title=Release v1.0 CreatedAt=2025-07-17T10:00:00Z
}

// 指针字段示例
type ExampleDocument struct {
	Title   string  `epack:"1"`
	Content *string `epack:"2"` // 可为 nil
	Version *int    `epack:"3"` // 可为 nil
}

func ExampleMarshal_pointer() {
	content := "Hello World"
	version := 3

	doc := ExampleDocument{
		Title:   "README",
		Content: &content,
		Version: &version,
	}

	data, _ := epack.Marshal(doc)

	var doc2 ExampleDocument
	epack.Unmarshal(data, &doc2)

	fmt.Printf("Title=%s Content=%s Version=%d\n",
		doc2.Title, *doc2.Content, *doc2.Version)

	// Output:
	// Title=README Content=Hello World Version=3
}

// map 类型示例
type ExampleConfig struct {
	Name       string            `epack:"1"`
	Properties map[string]string `epack:"2"`
}

func ExampleMarshal_map() {
	c := ExampleConfig{
		Name: "app-config",
		Properties: map[string]string{
			"host": "localhost",
			"port": "8080",
		},
	}

	data, _ := epack.Marshal(c)

	var c2 ExampleConfig
	epack.Unmarshal(data, &c2)

	fmt.Printf("Name=%s Properties=%v\n", c2.Name, c2.Properties)

	// Output:
	// Name=app-config Properties=map[host:localhost port:8080]
}

// LoadTemplate 预编译示例（提升重复序列化性能）
func ExampleLoadTemplate() {
	type Item struct {
		ID    int     `epack:"1"`
		Price float64 `epack:"2"`
	}

	// 预编译模板，后续 Marshal/Unmarshal 更快
	if err := epack.LoadTemplate(Item{}); err != nil {
		fmt.Println("load template error:", err)
		return
	}

	// 多次使用
	for i := 0; i < 3; i++ {
		item := Item{ID: i + 1, Price: float64(i+1) * 10.5}
		data, _ := epack.Marshal(item)

		var item2 Item
		epack.Unmarshal(data, &item2)
		fmt.Printf("ID=%d Price=%.1f\n", item2.ID, item2.Price)
	}

	// Output:
	// ID=1 Price=10.5
	// ID=2 Price=21.0
	// ID=3 Price=31.5
}
