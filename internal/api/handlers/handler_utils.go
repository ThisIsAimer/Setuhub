package handlers

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
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
		return fmt.Errorf("email format wrong")
	}
	return nil
}

func checkSection(section string) (string, error) {

	mySectrion := strings.TrimSpace(section)

	postType := make(map[string]string)

	postType["helpnearby"] = "Help near by"
	postType["impactevents"] = "Impact events"
	postType["moments"] = "Moments"
	postType["missingpeople"] = "Missing people"
	postType["bloodemergency"] = "Blood emergency"

	switch mySectrion {
	case "helpnearby", "impactevents", "moments", "missingpeople", "bloodemergency":
		return postType[section], nil
	default:
		return "", fmt.Errorf("invalid section: %s", section)
	}
}
