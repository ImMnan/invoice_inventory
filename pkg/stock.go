package pkg

import (
	"encoding/json"
	"io"
	"os"
)

func Stocks() ([]byte, error) {
	// open and read json
	var stock []TshirtStruct
	file, err := os.Open("Data/inventory.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &stock)
	if err != nil {
		return nil, err
	}

	// Return the entire stock data as JSON
	stockData, err := json.Marshal(stock)
	if err != nil {
		return nil, err
	}
	return stockData, nil
}
