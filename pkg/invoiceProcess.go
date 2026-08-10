package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	//"github.com/immnan/invoice_invoice/cmd/apply"
)

type Invoice struct {
	Type           string          `json:"type"`
	InvoiceID      string          `json:"invoice_id"`
	Customer       CustomerData    `json:"customer"`
	Date           string          `json:"date,omitempty"`
	IsPaid         bool            `json:"isPaid"`
	Product        []ProductStruct `json:"product"`
	Amount         int             `json:"amount"`
	TaxAmount      int             `json:"tax_amount"`
	PartialPayment int             `json:"partial_payment"`
}

type CustomerData struct {
	CustomerId string `json:"customer_id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Rating     int    `json:"rating"`
	GstNumber  string `json:"gst_number"`
	Contact    string `json:"contact"`
	Email      string `json:"email"`
	Origin     int    `json:"origin"`
}

func (data *JsLocalDB) Invoices() ([]byte, error) {
	// open and read json
	var invoices []Invoice
	file, err := os.Open(data.InvoiceFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(fileData, &invoices)
	if err != nil {
		return nil, err
	}

	// Return the entire invoice data as JSON
	invoiceData, err := json.Marshal(invoices)
	if err != nil {
		return nil, err
	}
	return invoiceData, nil
}

func (data *JsLocalDB) ProcessInvoiceWithStockData(product *ProductSlice) ([]string, error) {
	var invoicesSlice []string
	gst := 5

	// If any item is purchase or purchase-invoice, print success once and exit
	for _, item := range *product {
		t := strings.ToLower(item.Type)
		if t == "purchase" || t == "purchase-invoice" {
			fmt.Println("Purchase order added successfully, please check inventory or get purchase data.")
			return nil, nil
		}
	}

	filePath := data.InvoiceFile
	var existingInvoices []Invoice
	var invoices []Invoice
	if _, err := os.Stat(filePath); err == nil {
		f, err := os.Open(filePath)
		if err == nil {
			defer f.Close()
			json.NewDecoder(f).Decode(&existingInvoices)
		}
	}

	invoiceProductMap := make(map[string][]TshirtStruct)
	invoiceCustomerMap := make(map[string]string)
	// Group products by invoice ID
	for _, item := range *product {
		invoiceProductMap[item.Invoice] = append(invoiceProductMap[item.Invoice], item)
		if invoiceCustomerMap[item.Invoice] == "" {
			invoiceCustomerMap[item.Invoice] = item.For
		}
	}

	for invoiceID, filteredProducts := range invoiceProductMap {
		for _, oldInvoice := range existingInvoices {
			if oldInvoice.InvoiceID == invoiceID {
				return nil, fmt.Errorf("error: invoice %s already exists, terminating", invoiceID)
			}
		}
		customerID := invoiceCustomerMap[invoiceID]

		if customerID == "" {
			// Only skip with no warning if purchase type
			t := ""
			if len(filteredProducts) > 0 {
				t = strings.ToLower(filteredProducts[0].Type)
			}
			if t == "purchase" || t == "purchase-invoice" {
				continue
			}
			fmt.Printf("Warning: No customer ID found for invoice %s, skipping...\n", invoiceID)
			continue
		}

		customersData, err := data.getCustomerData()
		if err != nil {
			fmt.Println("Error fetching customer data:", err)
			return nil, err
		}
		var selectedCustomer *CustomerData
		for _, customer := range customersData {
			if customer.CustomerId == customerID {
				selectedCustomer = &customer
				break
			}
		}
		if selectedCustomer == nil {
			return nil, fmt.Errorf("customer with ID %s not found in customers.json", customerID)
		}
		var invoiceProducts []ProductStruct
		var totalAmount int
		productsData, err := data.getProductData()
		if err != nil {
			fmt.Println("Error fetching product data:", err)
			return nil, err
		}
		productMap := make(map[string]ProductStruct)
		for _, product := range productsData {
			productMap[product.ProductID] = product
		}
		for _, item := range filteredProducts {
			for _, prod := range item.Product {
				for color, quantities := range prod.Color {
					quantity := 0
					for _, qty := range quantities {
						quantity += qty
					}
					productInfo, exists := productMap[prod.ProductID]
					if !exists {
						return nil, fmt.Errorf("error: Product ID %s not found in products.json", prod.ProductID)
					}
					amount := int(prod.Price) * quantity
					totalAmount += amount
					invoiceProducts = append(invoiceProducts, ProductStruct{
						ProductID: prod.ProductID,
						Name:      productInfo.Name,
						Print:     prod.Print,
						Gen:       productInfo.Gen,
						GST:       gst,
						Color:     map[string][]int{color: quantities},
						Quantity:  quantity,
						Total:     amount,
						Price:     int(prod.Price),
					})
				}
			}
		}
		totalTax := 5 * totalAmount / 100
		invoice := Invoice{
			Type:      "Sales Invoice",
			InvoiceID: invoiceID,
			Customer:  *selectedCustomer,
			Date:      time.Now().Format("2006-01-02"),
			IsPaid:    false,
			Product:   invoiceProducts,
			Amount:    totalAmount,
			TaxAmount: totalTax,
		}
		invoices = append(invoices, invoice)
		invoicesSlice = append(invoicesSlice, invoice.InvoiceID)
	}
	existingInvoices = append(existingInvoices, invoices...)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(existingInvoices); err != nil {
		return nil, err
	}
	return invoicesSlice, nil
}

func (data *JsLocalDB) getProductData() ([]ProductStruct, error) {

	productDB := data.ProductFile

	filedata, err := os.Open(productDB)
	if err != nil {
		return nil, err
	}
	defer filedata.Close()

	var product []ProductStruct
	err = json.NewDecoder(filedata).Decode(&product)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (data *JsLocalDB) getCustomerData() ([]CustomerData, error) {

	customerDB := data.CustomerFile
	filedata, err := os.Open(customerDB)
	if err != nil {
		return nil, err
	}
	defer filedata.Close()

	var customers []CustomerData
	err = json.NewDecoder(filedata).Decode(&customers)
	if err != nil {
		return nil, err
	}
	return customers, nil
}
