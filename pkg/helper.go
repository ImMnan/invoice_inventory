package pkg

import (
	"encoding/csv"
	"fmt"
	"os"
)

type Proforma = TshirtStruct
type Purchase = TshirtStruct

type TshirtStruct struct {
	UUID     string        `json:"uuid"`
	Type     string        `json:"type"`
	For      string        `json:"for,omitempty"`
	From     string        `json:"from,omitempty"`
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

type ProBool struct {
	active bool
}

type PurBool struct {
	active bool
}

func ProcessApply(fileName string) ([][]string, error) {

	// Initialize the pointers before using them
	proforma := &ProBool{}
	purchase := &PurBool{}

	dataFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer dataFile.Close()

	dataValue, err := csv.NewReader(dataFile).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv data: %w", err)
	}
	var typeError []error
	var totalRows int
	for i, row := range dataValue {
		// Skip header row
		if i == 0 {
			continue
		}
		totalRows++
		switch row[1] {
		case "proforma":
			// Process proforma
			proforma.active = true
		case "purchase-invoice":
			// Process purchase
			purchase.active = true
		default:
			fmt.Printf("invalid data type: %s", row[1])
			typeError = append(typeError, fmt.Errorf("invalid data type at row %d: %s", i+1, row[1]))
			continue
		}

	}
	//proforma.processFileData(dataValue)
	//purchase.processFileData(dataValue)

	if len(typeError) > totalRows+1 {
		return nil, fmt.Errorf("encountered errors: %v", typeError)
	}
	return dataValue, nil
}

type ProcessData interface {
	processFileData(data [][]string) ([]byte, error)
}

type ApplyData interface {
	ApplyFileData(data [][]string, format string) error
}
