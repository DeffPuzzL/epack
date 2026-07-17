package epack

import (
	"encoding/json"
	"fmt"
	"testing"
)

type UserID int64
type Email string

type User struct {
	Profile `json:"metadata" epack:"1"`
	PF      *Profile `json:"preferences" epack:"2"`
	// Metadata map[string]any `json:"metadata" epack:"1"`
	// Profile *Profile `json:"profile,omitempty" epack:"1"` // 指针类型
	// Data    []byte   `json:"data" epack:"2"`              // 二进制数据
	// ID       UserID         `json:"id" epack:"1"`
	// Name     string         `json:"name" epack:"2"`
	// Email    Email          `json:"email" epack:"3"`
	// Age      int            `json:"age,omitempty" epack:"4"`
	// Height   float64        `json:"height" epack:"5"`
	// IsActive bool           `json:"is_active" epack:"6"`
	// Friends  []UserID       `json:"friends" epack:"7"`
	// Pre      Preferences    `json:"preferences" epack:"11"`      // 嵌套结构体
	// Class    interface{}    `json:"class,omitempty" epack:"12"`
	// CreatedAt time.Time      `json:"created_at" epack:"7"`
}

// 3. 嵌套结构体
type Profile struct {
	Biography string   `json:"bio" epack:"1"`
	Hobbies   []string `json:"hobbies" epack:"2"`
}

// 4. 接口类型
type Preferences interface {
	IsDefault() bool
}

// 5. 接口实现1
type BasicPreferences struct {
	Theme  string `json:"theme" epack:"1"`
	Locale string `json:"locale" epack:"2"`
}

func (b BasicPreferences) IsDefault() bool {
	return b.Theme == "light" && b.Locale == "en"
}

// 6. 接口实现2
type AdvancedPreferences struct {
	Theme     string   `json:"theme" epack:"1"`
	Locale    string   `json:"locale" epack:"2"`
	Features  []string `json:"features" epack:"3"`
	IsPremium bool     `json:"is_premium" epack:"4"`
}

func (a AdvancedPreferences) IsDefault() bool {
	return !a.IsPremium && len(a.Features) == 0
}

func TestParse(t *testing.T) {
	// 创建用户数据
	f := User{
		// ID:       1001,
		// Name:     "Alice Smith",
		// Email:    "alice@example.com",
		// Age:      30,
		// Height:   1.75,
		// IsActive: true,
		// // CreatedAt: time.Date(2023, time.January, 15, 14, 30, 0, 0, time.UTC),
		// Friends: []UserID{1002, 1003, 1004},
		Profile: Profile{
			Biography: "Software engineer passionate about Go",
			Hobbies:   []string{"reading", "hiking", "coding"},
		},
		PF: &Profile{
			Biography: "Software engineer passionate about Go",
			Hobbies:   []string{"reading", "hiking", "coding"},
		},
		// Metadata: map[string]any{
		// 	"last_login": time.Date(2023, time.March, 20, 10, 15, 0, 0, time.UTC),
		// "is_verified": true,
		// "scores":      []int{90, 85, 95},
		// },
		// Profile: &Profile{
		// 	Biography: "Software engineer passionate about Go",
		// 	Hobbies:   []string{"reading", "hiking", "coding"},
		// },
		// Data: []byte{0xDE, 0xAD, 0xBE, 0xEF}, // 二进制数据
		// Pre: AdvancedPreferences{
		// 	Theme:     "dark",
		// 	Locale:    "en-US",
		// 	Features:  []string{"notifications", "sync"},
		// 	IsPremium: true,
		// },
		// Class: map[string]interface{}{
		// 	"theme":      "dark",
		// 	"locale":     "en-US",
		// 	"features":   []string{"notifications", "sync"},
		// 	"is_premium": true,
		// },
	}

	if err := LoadTemplate(f); err != nil {
		fmt.Println("error", err)
		return
	}

	var err error
	var data []byte
	if data, err = Marshal(f); err != nil {
		fmt.Println("error", err)
		return
	}
	fmt.Println("data", len(data), data)

	n1 := new(User)
	// n1.Pre = new(AdvancedPreferences)
	if err = Unmarshal(data, n1); err != nil {
		fmt.Println("error", err)
		return
	}

	fmt.Println("\nOLD:", String(f))
	fmt.Println("\nNEW:", String(n1))
}

func String(v interface{}, format ...bool) string {
	var b []byte
	var err error
	if len(format) > 0 && format[0] {
		b, err = json.MarshalIndent(v, "", "  ")
	} else {
		b, err = json.Marshal(v)
	}
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}

	return string(b)
}
