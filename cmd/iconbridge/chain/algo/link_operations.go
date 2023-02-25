package algo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

const filename = "linkStatus.json"

func incrementField(fieldName string) error {
	// Read the contents of the file into a byte slice.
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into a map[string]interface{}.
	var data map[string]interface{}
	err = json.Unmarshal(fileBytes, &data)
	if err != nil {
		return err
	}

	// Increment the specified field.
	fieldValue, ok := data[fieldName].(uint64)
	if !ok {
		return fmt.Errorf("%s is not a float64", fieldName)
	}
	data[fieldName] = fieldValue + 1

	// Marshal the updated data back into a JSON string.
	updatedBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write the updated JSON string back to the file.
	err = ioutil.WriteFile(filename, updatedBytes, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
