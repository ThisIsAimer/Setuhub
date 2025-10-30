package main

import (
	"encoding/json"
	"fmt"
	"hackathon/internal/models"
)

func main() {

	var post models.Post

	fmt.Println("location", post.Location)
	if post.Location == nil {
		post.Location = []models.LocationObj{}
	}
	
	locJSON, err := json.Marshal(post.Location)
	if err != nil {
		fmt.Println("err", err)
		return
	}

	fmt.Println("bytes", string(locJSON))

}
