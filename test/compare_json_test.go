package test

import (
	"encoding/json"
	"testing"

	goccyjson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
)

//easyjson:json
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

var (
	data = []byte(`{"name":"John Doe","email":"john@example.com","age":30}`)
	user User
)

func BenchmarkEncodingJson(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := json.Unmarshal(data, &user); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJsoniter(b *testing.B) {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	for i := 0; i < b.N; i++ {
		if err := json.Unmarshal(data, &user); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoJson(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if err := goccyjson.Unmarshal(data, &user); err != nil {
			b.Fatal(err)
		}
	}
}

// func BenchmarkEasyjson(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		if err := user.UnmarshalJSON(data); err != nil {
// 			b.Fatal(err)
// 		}
// 	}
// }
