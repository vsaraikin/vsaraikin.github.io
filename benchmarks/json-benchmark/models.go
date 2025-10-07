package jsonbench

import "time"

//easyjson:json
type SimpleStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

//easyjson:json
type NestedStruct struct {
	ID      int                    `json:"id"`
	Name    string                 `json:"name"`
	Email   string                 `json:"email"`
	Active  bool                   `json:"active"`
	Score   float64                `json:"score"`
	Tags    []string               `json:"tags"`
	Meta    map[string]interface{} `json:"meta"`
	Created time.Time              `json:"created"`
}

//easyjson:json
type ComplexStruct struct {
	User     NestedStruct   `json:"user"`
	Friends  []NestedStruct `json:"friends"`
	Settings struct {
		Theme       string            `json:"theme"`
		Language    string            `json:"language"`
		Preferences map[string]string `json:"preferences"`
	} `json:"settings"`
	Metadata map[string]interface{} `json:"metadata"`
}

func generateSimpleData(n int) []SimpleStruct {
	data := make([]SimpleStruct, n)
	for i := 0; i < n; i++ {
		data[i] = SimpleStruct{
			ID:   i,
			Name: "User_" + string(rune(i)),
			Age:  20 + (i % 50),
		}
	}
	return data
}

func generateNestedData(n int) []NestedStruct {
	data := make([]NestedStruct, n)
	now := time.Now()
	for i := 0; i < n; i++ {
		data[i] = NestedStruct{
			ID:      i,
			Name:    "User_" + string(rune(i)),
			Email:   "user" + string(rune(i)) + "@example.com",
			Active:  i%2 == 0,
			Score:   float64(i) * 1.5,
			Tags:    []string{"tag1", "tag2", "tag3"},
			Meta:    map[string]interface{}{"key1": "value1", "key2": 123},
			Created: now,
		}
	}
	return data
}

func generateComplexData(n int) []ComplexStruct {
	data := make([]ComplexStruct, n)
	for i := 0; i < n; i++ {
		friends := make([]NestedStruct, 5)
		for j := 0; j < 5; j++ {
			friends[j] = NestedStruct{
				ID:    j,
				Name:  "Friend_" + string(rune(j)),
				Email: "friend" + string(rune(j)) + "@example.com",
				Tags:  []string{"friend", "social"},
			}
		}
		data[i] = ComplexStruct{
			User: NestedStruct{
				ID:    i,
				Name:  "User_" + string(rune(i)),
				Email: "user" + string(rune(i)) + "@example.com",
				Tags:  []string{"vip", "premium"},
			},
			Friends: friends,
			Metadata: map[string]interface{}{
				"version": "1.0",
				"source":  "benchmark",
			},
		}
		data[i].Settings.Theme = "dark"
		data[i].Settings.Language = "en"
		data[i].Settings.Preferences = map[string]string{"notifications": "on"}
	}
	return data
}
