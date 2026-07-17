package epack

import "fmt"

// Marshal / Unmarshal with epack field tags.
func ExampleMarshal() {
	type Person struct {
		Name string  `epack:"1"`
		Age  int     `epack:"2"`
		Tags []string `epack:"3"`
	}

	in := Person{Name: "Alice", Age: 30, Tags: []string{"go", "epack"}}
	data, err := Marshal(in)
	if err != nil {
		fmt.Println(err)
		return
	}

	var out Person
	if err := Unmarshal(data, &out); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", out)

	// Output:
	// {Name:Alice Age:30 Tags:[go epack]}
}

// LoadTemplate prebuilds the encode/decode plan for hot types.
func ExampleLoadTemplate() {
	type Item struct {
		ID    int     `epack:"1"`
		Price float64 `epack:"2"`
	}

	_ = LoadTemplate(Item{})

	data, _ := Marshal(Item{ID: 1, Price: 10.5})
	var item Item
	_ = Unmarshal(data, &item)
	fmt.Printf("ID=%d Price=%.1f\n", item.ID, item.Price)

	// Output:
	// ID=1 Price=10.5
}
