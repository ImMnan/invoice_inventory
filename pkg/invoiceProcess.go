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
	Type      string          `json:"type"`
	InvoiceID string          `json:"invoice_id"`
	Customer  CustomerData    `json:"customer"`
	Date      string          `json:"date,omitempty"`
	IsPaid    bool            `json:"isPaid"`
	Product   []ProductStruct `json:"product"`
	Amount    int             `json:"amount"`
	TaxAmount int             `json:"tax_amount"`
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

func Invoices() ([]byte, error) {
	// open and read json
	var invoices []Invoice
	file, err := os.Open("Data/invoices.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &invoices)
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

func (data *JsLocalDB) ProcessInvoiceWithStockData(format string, invoiceGroupedData *InvoiceGroupedData, product *ProductSlice) ([]string, error) {
	var invoices []Invoice
	var invoicesSlice []string
	gst := 5
	for invoiceID, invoiceItems := range invoiceGroupedData.SalesByInvoice {
		var customerID string
		if len(invoiceItems) > 0 {
			customerID = invoiceItems[0].For
		}
		if customerID == "" {
			fmt.Printf("Warning: No customer ID found for invoice %s, skipping...\n", invoiceID)
			continue
		}

		var stockUpdates map[string]map[string][]int
		if preCalculatedStock, exists := invoiceGroupedData.StockChangesByInvoice[invoiceID]; exists {
			stockUpdates = preCalculatedStock
		} else {
			stockUpdates = make(map[string]map[string][]int)
			for _, item := range invoiceItems {
				if stockUpdates[item.Product.ProductID] == nil {
					stockUpdates[item.Product.ProductID] = make(map[string][]int)
				}
				for color, quantities := range item.Product.Color {
					colorKey := strings.ToLower(color)
					if existing, exists := stockUpdates[item.Product.ProductID][colorKey]; exists {
						for i, qty := range quantities {
							if i < len(existing) {
								existing[i] += qty
							}
						}
					} else {
						stockUpdates[item.Product.ProductID][colorKey] = make([]int, len(quantities))
						copy(stockUpdates[item.Product.ProductID][colorKey], quantities)
					}
				}
			}
		}

		printMap := make(map[string]string)
		priceMap := make(map[string]float64)
		for _, item := range *product {
			printMap[item.Product.ProductID] = item.Product.Print
			priceMap[item.Product.ProductID] = float64(item.Product.Price)
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

		productsData, err := data.getProductData()
		if err != nil {
			fmt.Println("Error fetching product data:", err)
			return nil, err
		}
		productMap := make(map[string]ProductStruct)
		for _, product := range productsData {
			productMap[product.ProductID] = product
		}

		// Build ProductStruct slice for this invoice
		var invoiceProducts []ProductStruct
		var totalAmount int
		for productId, colors := range stockUpdates {
			productInfo, exists := productMap[productId]
			if !exists {
				return nil, fmt.Errorf("error: Product ID %s not found in products.json", productId)
			}
			for color, quantities := range colors {
				quantity := 0
				for _, qty := range quantities {
					quantity += qty
				}
				price := priceMap[productId]
				if price == 0 {
					price = float64(productInfo.Price)
				}
				amount := int(price * float64(quantity))
				totalAmount += amount
				// Tax is now calculated on the totalAmount after all products are processed
				invoiceProducts = append(invoiceProducts, ProductStruct{
					ProductID: productId,
					Name:      productInfo.Name,
					Print:     printMap[productId],
					Gen:       productInfo.Gen,
					GST:       gst, // GST per product not calculated here
					Color:     map[string][]int{color: quantities},
					Quantity:  quantity,
					Total:     totalAmount,
					Price:     int(price),
				})
			}
		}
		// Calculate total tax on the totalAmount after all products are processed
		totalTax := 5 * totalAmount / 100
		invoice := Invoice{
			Type:      "Sales Invoice",
			InvoiceID: invoiceID,
			Customer:  *selectedCustomer,
			Date:      time.Now().Format("2006-01-02"),
			IsPaid:    false, // set as needed
			Product:   invoiceProducts,
			Amount:    totalAmount,
			TaxAmount: totalTax,
		}
		invoices = append(invoices, invoice)
		invoicesSlice = append(invoicesSlice, invoice.InvoiceID)

	}

	// Write or append to Data/invoices.json
	filePath := "Data/invoices.json"
	var existingInvoices []Invoice
	if _, err := os.Stat(filePath); err == nil {
		f, err := os.Open(filePath)
		if err == nil {
			defer f.Close()
			json.NewDecoder(f).Decode(&existingInvoices)
		}
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
