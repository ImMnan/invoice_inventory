package pkg

import (
	"encoding/json"
	"fmt"
)

type Invoice struct {
	Type    string        `json:"type"`
	For     string        `json:"for,omitempty"`
	Invoice string        `json:"invoice"`
	Date    string        `json:"date,omitempty"`
	IsPaid  bool          `json:"isPaid"`
	Product ProductStruct `json:"product"`
}

func makeInvoice(proformaData []byte) {
	// Create a new invoice based on the proforma data
	var proforma Proforma
	err := json.Unmarshal(proformaData, &proforma)
	if err != nil {
		fmt.Println("Error parsing proforma data:", err)
		return
	}

	invoice := Invoice{
		Type:    "Invoice",
		For:     proforma.For,
		Invoice: proforma.Invoice,
		Date:    proforma.Date,
		IsPaid:  proforma.IsPaid,
		Product: proforma.Product,
	}

	// Convert the Json to csv format

	// Save the invoice to a file or database (not implemented here)
	fmt.Println("Invoice created successfully:", string(invoiceData))

}
