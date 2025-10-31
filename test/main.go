package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func main() {

	type Post struct {
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"created_at"`
	}

	p := Post{
		Title:     "Test",
		CreatedAt: time.Now(),
	}

	b, _ := json.Marshal(p)
	fmt.Println(string(b))

}
