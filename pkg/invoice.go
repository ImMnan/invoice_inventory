package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

type Invoice struct {
	Type    string        `json:"type"`
	For     string        `json:"for,omitempty"`
	Invoice string        `json:"invoice"`
	Date    string        `json:"date,omitempty"`
	IsPaid  bool          `json:"isPaid"`
	Product ProductStruct `json:"product"`
}

type ProductData struct {
	ProductId   string              `json:"product_id"`
	Name        string              `json:"name"`
	Colors      map[string][]string `json:"colors"`
	Description string              `json:"description"`
	Price       float64             `json:"price"`
	Print       string              `json:"print"`
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

func (product *ProductSlice) makeInvoiceWithStockData(format string, invoiceGroupedData *InvoiceGroupedData) error {

	// Generate invoices grouped by invoice ID
	for invoiceID, invoiceItems := range invoiceGroupedData.SalesByInvoice {
		//fmt.Printf("\n=== Generating invoice: %s ===\n", invoiceID)
		// Extract customer ID from the first item in this invoice
		var customerID string
		if len(invoiceItems) > 0 {
			customerID = invoiceItems[0].For
		}

		if customerID == "" {
			fmt.Printf("Warning: No customer ID found for invoice %s, skipping...\n", invoiceID)
			continue
		}

		// Use pre-calculated stock changes instead of recalculating
		var stockUpdates map[string]map[string][]int
		if preCalculatedStock, exists := invoiceGroupedData.StockChangesByInvoice[invoiceID]; exists {
			// Use the pre-calculated stock data from inventory update process
			stockUpdates = preCalculatedStock
		} else {
			// Fallback: calculate stockUpdates if not provided (for backward compatibility)
			stockUpdates = make(map[string]map[string][]int)
			for _, item := range invoiceItems {
				// Initialize stock updates for this product if not exists
				if stockUpdates[item.Product.UID] == nil {
					stockUpdates[item.Product.UID] = make(map[string][]int)
				}

				// Track quantities for this invoice
				for color, quantities := range item.Product.Color {
					colorKey := strings.ToLower(color)
					if existing, exists := stockUpdates[item.Product.UID][colorKey]; exists {
						// Add to existing quantities
						for i, qty := range quantities {
							if i < len(existing) {
								existing[i] += qty
							}
						}
					} else {
						// Initialize with current quantities
						stockUpdates[item.Product.UID][colorKey] = make([]int, len(quantities))
						copy(stockUpdates[item.Product.UID][colorKey], quantities)
					}
				}
			}
		}

		//var proformaItems []Proforma
		//if err := json.Unmarshal(proformaData, &proformaItems); err != nil {
		//	return fmt.Errorf("failed to parse proforma data: %v", err)
		//}
		// Create a map to lookup print and price data from proforma items
		printMap := make(map[string]string)
		priceMap := make(map[string]float64)
		for _, item := range *product {
			printMap[item.Product.UID] = item.Product.Print
			priceMap[item.Product.UID] = float64(item.Product.Price)
		}

		// Load customer data first to validate
		customersData, err := getCustomerData("Data/customers.json")
		if err != nil {
			fmt.Println("Error fetching customer data:", err)
			return err
		}

		// Validate customer ID against customers.json
		var selectedCustomer *CustomerData
		for _, customer := range customersData {
			if customer.CustomerId == customerID {
				selectedCustomer = &customer
				break
			}
		}

		if selectedCustomer == nil {
			return fmt.Errorf("customer with ID %s not found in customers.json", customerID)
		}

		// Load products data for name and price lookup
		productsData, err := getProductData("Data/products.json")
		if err != nil {
			fmt.Println("Error fetching product data:", err)
			return err
		}

		// Create a map for quick product lookup by ID
		productMap := make(map[string]ProductData)
		for _, product := range productsData {
			productMap[product.ProductId] = product
		}

		if format == "table" {
			fmt.Println("\nFROM: SHIRIKRISHNA TECH\nGST: 1234567890\nCOO: INDIA\nCONTACT: 9725359497\n---")
			invoiceTab := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			// Print invoice header
			fmt.Fprintln(invoiceTab, "\nTYPE\tINVOICE\tNAME\tADDRESS\tGST NUMBER\tDATE")
			fmt.Fprintf(invoiceTab, "%s\t%s\t%s\t%s\t%s\t%s\n\n", "Sales Invoice",
				invoiceID,
				selectedCustomer.Name,
				selectedCustomer.Address,
				selectedCustomer.GstNumber,
				time.Now().Format("2006-01-02"))
			invoiceTab.Flush()
			//	customerTab := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			// Print customer details first
			//	fmt.Fprintln(customerTab, "\nNAME\tADDRESS\tGST NUMBER")
			//	fmt.Fprintf(customerTab, "%s\t%s\t%s\n\n",
			//		selectedCustomer.Name,
			//		selectedCustomer.Address,
			//		selectedCustomer.GstNumber)

			//			customerTab.Flush()

			// Use tabwriter for aligned table output
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

			// Print product header
			fmt.Fprintln(w, "PRODUCT ID\tPRODUCT Name\tPRICE\tPRINT\tCOLOR\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTOTAL\tAMOUNT")
			fmt.Fprintln(w, "----------\t------------\t-----\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t----\t----")

			// Process each product in stockUpdates
			for productId, colors := range stockUpdates {
				// Look up product name and price
				productInfo, exists := productMap[productId]
				if !exists {
					return fmt.Errorf("error: Product ID %s not found in products.json", productId)
				}

				// Process each color for this product
				for color, quantities := range colors {
					// Calculate total quantity
					total := 0
					for _, qty := range quantities {
						total += qty
					}

					// Calculate amount (price × total)
					// Get price data from proforma
					price := priceMap[productId]
					if price == 0 {
						// Fallback to products.json if no price in proforma
						price = productInfo.Price
					}
					amount := price * float64(total)

					// Get print data from proforma
					printValue := printMap[productId]
					if printValue == "" {
						printValue = "N/A" // fallback if no print data
					}

					// Format the line with tabs for tabwriter
					line := fmt.Sprintf("%s\t%s\t%.0f\t%s\t%s",
						productId,
						productInfo.Name,
						price,
						printValue,
						color,
					)

					// Add size quantities
					for _, qty := range quantities {
						line += fmt.Sprintf("\t%d", qty)
					}

					// Add total and amount
					line += fmt.Sprintf("\t%d\t%.0f", total, amount)

					fmt.Fprintln(w, line)
				}
			}

			// Flush the tabwriter to display the formatted table
			w.Flush()

		}
		if format == "csv" {
			// CSV format output
			fmt.Println("Product ID,Product Name,Price,PRINT,Color,XS,S,M,L,XL,2XL,3XL,4XL,Total,Amount")

			// Process each product in stockUpdates
			for productId, colors := range stockUpdates {
				// Look up product name and price
				productInfo, exists := productMap[productId]
				if !exists {
					return fmt.Errorf("error: Product ID %s not found in products.json", productId)
				}

				// Process each color for this product
				for color, quantities := range colors {
					// Calculate total quantity
					total := 0
					for _, qty := range quantities {
						total += qty
					}

					// Calculate amount (price × total)
					// Get price data from proforma
					price := priceMap[productId]
					if price == 0 {
						// Fallback to products.json if no price in proforma
						price = productInfo.Price
					}
					amount := price * float64(total)

					// Get print data from proforma
					printValue := printMap[productId]
					if printValue == "" {
						printValue = "N/A" // fallback if no print data
					}

					// Format the line as CSV
					line := fmt.Sprintf("%s,%s,%.0f,%s,%s",
						productId,
						productInfo.Name,
						price,
						printValue,
						color)

					// Add size quantities
					for _, qty := range quantities {
						line += fmt.Sprintf(",%d", qty)
					}

					// Add total and amount
					line += fmt.Sprintf(",%d,%.0f", total, amount)

					fmt.Println(line)
				}
			}
		}

		// Following is the output I need to be passed to a txt file.
		// SHRIKRISHNA TECH, GST: 1234567890, India

		// Type, Invoice, Date
		// Invoice, SKCP-001, 2023-01-01

		// Name, Address, Gst Number, Country, Date
		// RK Tees, Navsari, 1234567890, India, 2023-01-01

		// Product ID, Product Name, Price, Color, XS, S, M, L, XL, 2XL, 3XL, 4XL, Total, Amount
		// CO180G-RG, 100% Cotton | T-Shirt | Bio-Washed, 200, Red, 1, 5, 5, 5, 5, 5, 1, 1, 28, 5600

		// if tax invoice
		// iGST, sGST, GstTotal
		// 2.5%, 2.5%, 5% of Amount (i.e 280)

		// TaxInvoice Total amount, Payment Terms, Payment Mode
		// 5880, Due on Receipt, Online Transfer/Cash

	}
	return nil
}

func getProductData(fileName string) ([]ProductData, error) {

	filedata, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer filedata.Close()

	var product []ProductData
	err = json.NewDecoder(filedata).Decode(&product)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func getCustomerData(fileName string) ([]CustomerData, error) {
	filedata, err := os.Open(fileName)
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

// makeInvoice - Original function for backward compatibility
func (product *ProductSlice) makeInvoice(format string, invoiceGroups map[string][]Proforma) error {
	// Create InvoiceGroupedData structure for the enhanced function
	invoiceGroupedData := &InvoiceGroupedData{
		SalesByInvoice:        invoiceGroups,
		StockChangesByInvoice: nil, // Will trigger fallback calculation in makeInvoiceWithStockData
	}

	// Call the enhanced version
	return product.makeInvoiceWithStockData(format, invoiceGroupedData)
}
