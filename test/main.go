package main

import (
	"fmt"
	"hackathon/internal/models"
	"reflect"
)

func checkEmptyField(modle any) error {
	modleValue := reflect.ValueOf(modle)
	modleType := modleValue.Type()

	for i := range modleType.NumField() {

		if modleValue.Field(i).String() == "" {
			return fmt.Errorf("empty fields found")
		}

	}

	return nil
}
func main() {

	userInfo := models.UserInfo{Aadhar: "w", Phone: "32", Gender: "", Address: "yes"}

	err := checkEmptyField(userInfo)

	if err != nil {
		fmt.Println("error is:", err)
		return
	}
	fmt.Println(" fields full")

}
