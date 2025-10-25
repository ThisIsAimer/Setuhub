package handlers

import (
	"fmt"
	"reflect"
)

func checkEmptyField(modle any) error {
	modleValue := reflect.ValueOf(modle)
	modleType := modleValue.Type()

	for i := range modleType.NumField() {
		dbTag := modleType.Field(i).Tag.Get("db")

		if modleValue.Field(i).String() == "" {
			return fmt.Errorf("empty fields found %s", dbTag)
		}

	}

	return nil
}
