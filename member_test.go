package epack

import (
	"fmt"
	"testing"
)

func TestMember(t *testing.T) {
	// 创建用户数据
	f := map[string]interface{}{
		"Biography": "Software engineer passionate about Go",
		"Hobbies":   []string{"reading", "hiking", "coding"},
		"Time":      1231421,
	}

	var err error
	var data []byte
	if data, err = Marshal(f); err != nil {
		fmt.Println("error", err)
		return
	}
	fmt.Println("data", len(data), data)

	n1 := make(map[string]interface{})
	if err = Unmarshal(data, &n1); err != nil {
		fmt.Println("error", err)
		return
	}

	fmt.Println("\nOLD:", String(f))
	fmt.Println("\nNEW:", String(n1))
}
