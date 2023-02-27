package algo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/icon-project/icon-bridge/cmd/iconbridge/chain"
)

const filePath = "chain/algo/linkStatus.json"

// create a function similar to the next one, but to replace the field by an argument float value instead of increment it
func updateField(fieldName string, value ...uint64) error {
	// Read the contents of the file into a byte slice.
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into a map[string]interface{}.
	var data map[string]interface{}
	err = json.Unmarshal(fileBytes, &data)
	if err != nil {
		return err
	}

	// Check if the field exists in the data.
	if _, ok := data[fieldName]; !ok {
		return fmt.Errorf("%s field not found", fieldName)
	}

	// If there's only one value passed, increment the field.
	if len(value) == 0 {
		fieldValue, ok := data[fieldName].(float64)
		if !ok {
			return fmt.Errorf("%s is not a float64", fieldName)
		}
		data[fieldName] = fieldValue + 1
	} else if len(value) == 1 {
		// If there's two values passed, replace the field value with the new value.
		data[fieldName] = value[0]
	} else {
		return fmt.Errorf("too many arguments")
	}

	// Marshal the updated data back into a JSON string.
	updatedBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write the updated JSON string back to the file.
	err = ioutil.WriteFile(filePath, updatedBytes, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func getStatus() (*chain.BMCLinkStatus, error) {
	f, err := os.Open("chain/algo/linkStatus.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	link := &bmcLink{}
	if err := json.NewDecoder(f).Decode(&link); err != nil {
		return nil, err
	}

	return &chain.BMCLinkStatus{
		TxSeq:         link.TxSeq,
		RxSeq:         link.RxSeq,
		RxHeight:      link.RxHeight,
		CurrentHeight: link.TxHeight,
	}, nil
}
