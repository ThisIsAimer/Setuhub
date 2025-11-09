package handlers

import (
	"fmt"
	"hackathon/pkg/utils"
	"net/http"
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

func isValidEmailFormat(email string) utils.Errorhandler {
	var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return utils.ErrorHandler(fmt.Errorf("email format wrong"), "invalid email", http.StatusBadRequest)
	}
	return utils.Errorhandler{}
}

func checkSection(section string) (string, utils.Errorhandler) {

	mySectrion := strings.TrimSpace(section)

	postType := make(map[string]string)

	postType["helpnearby"] = "Help near by"
	postType["impactevents"] = "Impact events"
	postType["moments"] = "Moments"
	postType["missingpeople"] = "Missing people"
	postType["bloodemergency"] = "Blood emergency"

	switch mySectrion {
	case "helpnearby", "impactevents", "moments", "missingpeople", "bloodemergency":
		return postType[section], utils.Errorhandler{}
	default:
		return "", utils.ErrorHandler(fmt.Errorf("invalid section: %s", section), "invalid route", http.StatusBadRequest)
	}
}
