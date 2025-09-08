package pkg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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

func (data *JsLocalDB) ProcessInvoiceWithStockData(invoiceGroupedData *InvoiceGroupedData, product *ProductSlice) ([]string, error) {
	var invoicesSlice []string
	gst := 5
	// Write or append to Data/invoices.json
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

	for invoiceID, invoiceItems := range invoiceGroupedData.SalesByInvoice {
		// Only use product rows for this invoiceID
		var filteredProducts []TshirtStruct
		for _, item := range *product {
			if item.Invoice == invoiceID {
				filteredProducts = append(filteredProducts, item)
			}
		}
		// Debug print for filteredProducts
		//fmt.Printf("\n[DEBUG] InvoiceID: %s, filteredProducts count: %d\n", invoiceID, len(filteredProducts))
		//for i, item := range filteredProducts {
		//fmt.Printf("[DEBUG] %d: Invoice=%s, ProductID=%s, Print=%s, Color=%v, Qty=%d\n", i, item.Invoice, item.Product.ProductID, item.Product.Print, item.Product.Color, item.Product.Quantity)
		for _, oldInvoice := range existingInvoices {
			if oldInvoice.InvoiceID == invoiceID {
				return nil, fmt.Errorf("error: invoice %s already exists, terminating", invoiceID)
			}
		}
		var customerID string
		if len(invoiceItems) > 0 {
			customerID = invoiceItems[0].For
		}
		if customerID == "" {
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
		// Build ProductStruct slice for this invoice: each proforma row becomes a separate item
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
			for color, quantities := range item.Product.Color {
				quantity := 0
				for _, qty := range quantities {
					quantity += qty
				}
				productInfo, exists := productMap[item.Product.ProductID]
				if !exists {
					return nil, fmt.Errorf("error: Product ID %s not found in products.json", item.Product.ProductID)
				}
				amount := int(item.Product.Price) * quantity
				totalAmount += amount
				invoiceProducts = append(invoiceProducts, ProductStruct{
					ProductID: item.Product.ProductID,
					Name:      productInfo.Name,
					Print:     item.Product.Print,
					Gen:       productInfo.Gen,
					GST:       gst, // GST per product not calculated here
					Color:     map[string][]int{color: quantities},
					Quantity:  quantity,
					Total:     amount,
					Price:     int(item.Product.Price),
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

func SimpleInvoice(data *JsLocalDB) error {
	// This function reads the CSV, groups by invoice ID, and prints invoices in invoices.json format (no calculations, no file write)

	fileData, err := os.Open(data.InvoiceFile)
	if err != nil {
		return err
	}
	defer fileData.Close()

	dataValue, err := csv.NewReader(fileData).ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read csv data: %w", err)
	}

	// Group rows by invoice ID
	type invoiceGroup struct {
		rows [][]string
	}
	groups := make(map[string]*invoiceGroup)

	for i, row := range dataValue {
		if i == 0 {
			continue // skip header
		}
		if row[0] != "proforma" {
			continue
		}
		invoiceID := strings.TrimSpace(row[1])
		if invoiceID == "" {
			continue
		}
		if _, ok := groups[invoiceID]; !ok {
			groups[invoiceID] = &invoiceGroup{}
		}
		groups[invoiceID].rows = append(groups[invoiceID].rows, row)
	}

	var invoices []Invoice

	for invoiceID, group := range groups {
		// Use the first row for customer info, date, etc.
		row := group.rows[0]
		customerID := strings.TrimSpace(row[2])
		// Build CustomerData (minimal, as CSV may not have all fields)
		customer := CustomerData{
			CustomerId: customerID,
			Name:       "",
			Address:    "",
			Rating:     0,
			GstNumber:  "",
			Contact:    "",
			Email:      "",
			Origin:     0,
		}
		var products []ProductStruct
		for _, row := range group.rows {
			if len(row) < 19 {
				continue
			}
			colorMap := make(map[string][]int)
			quantities := make([]int, 8)
			for j := 0; j < 8; j++ {
				if qty, err := strconv.Atoi(strings.TrimSpace(row[10+j])); err == nil {
					quantities[j] = qty
				}
			}
			color := strings.TrimSpace(row[9])
			colorMap[color] = quantities
			quantity := 0
			if qtyStr := strings.TrimSpace(row[18]); qtyStr != "" {
				if t, err := strconv.Atoi(qtyStr); err == nil {
					quantity = t
				}
			}
			priceInt := 0
			if p, err := strconv.Atoi(strings.TrimSpace(row[5])); err == nil {
				priceInt = p
			}
			gstInt := 0
			if g, err := strconv.Atoi(strings.TrimSpace(row[6])); err == nil {
				gstInt = g
			}
			products = append(products, ProductStruct{
				ProductID: strings.TrimSpace(row[3]),
				Name:      "",
				Print:     strings.TrimSpace(row[7]),
				Gen:       strings.TrimSpace(row[8]),
				GST:       gstInt,
				Color:     colorMap,
				Quantity:  quantity,
				Total:     priceInt * quantity,
				Price:     priceInt,
			})
		}
		invoice := Invoice{
			Type:      "Sales Invoice",
			InvoiceID: invoiceID,
			Customer:  customer,
			Date:      strings.TrimSpace(row[4]),
			IsPaid:    false,
			Product:   products,
			Amount:    0, // No calculation
			TaxAmount: 0, // No calculation
		}
		invoices = append(invoices, invoice)
	}

	jsonData, err := json.MarshalIndent(invoices, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}
