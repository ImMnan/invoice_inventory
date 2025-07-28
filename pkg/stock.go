package pkg

import (
	"encoding/json"
	"io"
	"os"
)

type TshirtStruct struct {
	ID        string      `json:"id,omitempty"`
	ProductID string      `json:"product_id"`
	Type      string      `json:"type"`
	Invoice   string      `json:"invoice"`
	Created   string      `json:"date"`
	Gst       string      `json:"gst"`
	Print     string      `json:"print"`
	Color     colorStruct `json:"color"`
	Total     int         `json:"total"`
	Rejected  bool        `json:"rejected"`
	IsPaid    bool        `json:"isPaid"`
	Gen       string      `json:"gen"`
}

type colorStruct map[string][]int

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
