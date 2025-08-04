package pkg

import (
	"encoding/json"
	"io"
	"os"
)

type TshirtStruct struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	Invoice  string        `json:"invoice"`
	Date     string        `json:"date,omitempty"`
	IsPaid   bool          `json:"isPaid"`
	Rejected bool          `json:"rejected"`
	Product  ProductStruct `json:"product"`
}

type ProductStruct struct {
	UID   string           `json:"uid"`
	Print string           `json:"print"`
	Gen   string           `json:"gen"`
	GST   string           `json:"gst"`
	Color map[string][]int `json:"color"`
	Total int              `json:"total"`
	Price int              `json:"price"`
}

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
