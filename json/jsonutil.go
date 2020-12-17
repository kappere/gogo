package json

import (
	"encoding/json"

	"wataru.com/gogo/logger"
)

func ToJson(data interface{}) string {
	return string(ToJsonByte(data))
}

func ToJsonByte(data interface{}) []byte {
	jsons, errs := json.Marshal(data)
	if errs != nil {
		logger.Error(errs.Error())
	}
	return jsons
}

func FromJson(data string, target interface{}) {
	FromJsonByte([]byte(data), target)
}

func FromJsonByte(data []byte, target interface{}) {
	json.Unmarshal([]byte(data), target)
}
