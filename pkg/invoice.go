package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
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

func makeInvoice(stockUpdates map[string]map[string][]int, format string, customerID string) error {
	// Create a new invoice based on the stock updates data
	// format can be "table" or "csv"

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
		customerTab := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		// Print customer details first
		fmt.Fprintln(customerTab, "Name", "Address", "GST Number")
		fmt.Fprintf(customerTab, "%s\t%s\t%s\n\n",
			selectedCustomer.Name,
			selectedCustomer.Address,
			selectedCustomer.GstNumber)
		customerTab.Flush()

		// Use tabwriter for aligned table output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

		// Print product header
		fmt.Fprintln(w, "Product ID\tProduct Name\tPrice\tColor\tXS\tS\tM\tL\tXL\t2X\t3X\t4X\tTotal\tAmount")
		fmt.Fprintln(w, "----------\t------------\t-----\t-----\t--\t--\t--\t--\t--\t--\t--\t--\t----\t----")

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
				amount := productInfo.Price * float64(total)

				// Format the line with tabs for tabwriter
				line := fmt.Sprintf("%s\t%s\t%.0f\t%s",
					productId,
					productInfo.Name,
					productInfo.Price,
					color)

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
		fmt.Println("Product ID,Product Name,Price,Color,XS,S,M,L,XL,2XL,3XL,4XL,Total,Amount")

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
				amount := productInfo.Price * float64(total)

				// Format the line as CSV
				line := fmt.Sprintf("%s,%s,%.0f,%s",
					productId,
					productInfo.Name,
					productInfo.Price,
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
