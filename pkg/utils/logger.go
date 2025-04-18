package utils

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sirupsen/logrus"
)

func FormatKeyValueLogs(data [][2]string) string {
	var builder strings.Builder
	builder.Grow(len(data) * 10)

	for _, entry := range data {
		builder.WriteString(fmt.Sprintf("  %s: %s\n", entry[0], entry[1]))
	}

	return builder.String()
}

func LogInfo(title string, message string) {
	if logrus.GetLevel() < logrus.InfoLevel {
		return
	}

	logrus.Info(fmt.Sprintf(
		"\033[1m%s\033[0m:\n%s",
		title,
		message,
	))
}

func LogError(message string, errStr string) {
	logrus.Error(fmt.Sprintf(
		"%s: \033[38;5;197m%s\033[0m",
		message,
		errStr,
	))
}

func LogResponse(url string, response interface{}) {
	if logrus.GetLevel() < logrus.InfoLevel {
		return
	}

	keyValueData := ParseStructToKeyValue(response, "")
	message := FormatKeyValueLogs(keyValueData)

	logrus.Info(fmt.Sprintf(
		"URL request: \033[1m%s\033[0m:\n%s",
		url,
		message,
	))
}

func ParseStructToKeyValue(response interface{}, prefix string) [][2]string {
	var keyValuePairs [][2]string

	val := reflect.ValueOf(response)
	if val.Kind() == reflect.Ptr {
		val = val.Elem() // Dereference pointer if necessary.
	}

	if val.Kind() != reflect.Struct {
		return keyValuePairs // Return empty for non-struct types.
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldName := field.Name
		fieldValue := val.Field(i)

		// Construct the full key name (including prefix for nested structs).
		fullName := fieldName
		if prefix != "" {
			fullName = fmt.Sprintf("%s.%s", prefix, fieldName)
		}

		// Check if the field is a nested struct.
		if fieldValue.Kind() == reflect.Struct {
			// Recursively process the nested struct.
			nestedKeyValuePairs := ParseStructToKeyValue(fieldValue.Interface(), fullName)
			keyValuePairs = append(keyValuePairs, nestedKeyValuePairs...)
		} else {
			// Convert the field value to a string representation.
			fieldValueStr := fmt.Sprintf("%v", fieldValue.Interface())
			keyValuePairs = append(keyValuePairs, [2]string{fullName, fieldValueStr})
		}
	}

	return keyValuePairs
}
