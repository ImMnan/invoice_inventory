package pkg

import (
	"encoding/json"
	"io"
	"os"
)

func (data *JsLocalDB) Customers() ([]byte, error) {
	// open and read json
	var customers []Customers
	file, err := os.Open(data.CustomerFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fileData, &customers)
	if err != nil {
		return nil, err
	}

	// Return the entire stock data as JSON
	customerData, err := json.Marshal(customers)
	if err != nil {
		return nil, err
	}
	return customerData, nil
}
