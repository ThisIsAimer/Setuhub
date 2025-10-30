package handlers

import (
	"fmt"
	"hackathon/pkg/utils"
	"reflect"
	"regexp"
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

func isValidEmailFormat(email string) error {
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return utils.ErrorHandler(fmt.Errorf("email format wrong"), "invalid email")
	}
	return nil
}

func checkSection(section string) error {
	switch section {
	case "help", "event", "media", "missing", "blood":
		return nil
	default:
		return utils.ErrorHandler(fmt.Errorf("invalid section: %s", section), "invalid route")
	}
}
