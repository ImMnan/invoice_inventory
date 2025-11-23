package get

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

var customersCmd = &cobra.Command{
	Use:     "customers [customer_name] [customer_id]",
	Short:   "Get customer details (name and/or ID)",
	Aliases: []string{"customer", "cs"},
	Long: `Fetch customer information.
Arguments:
  [customer_name]  (optional) First argument treated as name if it does not look like an ID.
  [customer_id]    (optional) Second argument treated as ID.
If only one argument is provided and it looks like an ID (e.g. starts with SK or contains a dash), it is treated as the ID.
Use 'all' to skip filtering.
Examples:
  lvs get customers                # all customers
  lvs get customers "RK Tees"       # by name
  lvs get customers SK21-001       # by ID
  lvs get customers "RK Tees" SK21-001  # by name AND ID
`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		name := "all"
		id := "all"
		if len(args) == 1 {
			if looksLikeID(args[0]) {
				id = args[0]
			} else {
				name = args[0]
			}
		} else if len(args) >= 2 {
			name = args[0]
			id = args[1]
		} else if len(args) == 0 {
			// defaults already set
		}
		customerAddress, _ := cmd.Flags().GetString("address")
		rating, _ := cmd.Flags().GetInt("rating")
		PrintCustomers(name, id, customerAddress, rating)
	},
}

// looksLikeID provides a heuristic to decide if a single arg is an ID.
func looksLikeID(arg string) bool {
	upper := strings.ToUpper(arg)
	if strings.HasPrefix(upper, "SK") { // project-specific prefix
		return true
	}
	if strings.Contains(arg, "-") { // typical ID pattern with dash
		return true
	}
	return false
}

func init() {
	GetCmd.AddCommand(customersCmd)
	customersCmd.Flags().StringP("address", "a", "all", "Filter customers by address")
	customersCmd.Flags().Int("rating", 0, "Filter customers by rating")
}

func PrintCustomers(customerName, customerID, customerAddress string, rating int) {
	inventoryDB, customerDB, invoiceDB, productDB, err := ConfigData(0)
	if err != nil {
		fmt.Println("Error fetching config data:", err)
		return
	}
	existData := &pkg.JsLocalDB{
		InventoryFile: inventoryDB,
		CustomerFile:  customerDB,
		InvoiceFile:   invoiceDB,
		ProductFile:   productDB,
	}

	data, err := existData.Customers()
	if err != nil {
		fmt.Println("Error fetching customer data:", err)
		return
	}
	var customers []pkg.Customers
	json.Unmarshal(data, &customers)

	// Create filter based on input parameters
	filter := CustomerFilter{
		CustomerName: customerName,
		CustomerID:   customerID,
		Address:      customerAddress,
		Rating:       rating,
	}

	customerWe := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(customerWe, "CUSTOMER\tID\tRATE\tPHONE\tADDRESS\tORIGIN")

	for _, cust := range customers {
		if !filter.shouldShowCustomer(cust) {
			continue
		}
		fmt.Fprintf(customerWe, "%s\t%s\t%d\t%s\t%s\t%d\n",
			cust.DisplayName(),
			cust.CustomerID,
			cust.Rating,
			cust.Phone,
			cust.Address,
			cust.Origin)
	}

	// Ensure buffered output is written
	customerWe.Flush()

}

func (cf CustomerFilter) shouldShowCustomer(cust pkg.Customers) bool {
	nameFilter := strings.ToLower(strings.TrimSpace(cf.CustomerName))
	idFilter := strings.ToLower(strings.TrimSpace(cf.CustomerID))
	addrFilter := strings.ToLower(strings.TrimSpace(cf.Address))
	// Case-insensitive comparisons; treat any variant of 'all' as no filter.
	custName := cust.DisplayName()
	if nameFilter != "all" && strings.ToLower(custName) != nameFilter {
		return false
	}
	if idFilter != "all" && strings.ToLower(cust.CustomerID) != idFilter {
		return false
	}
	if addrFilter != "all" && strings.ToLower(cust.Address) != addrFilter {
		return false
	}
	if cf.Rating != 0 && cust.Rating != cf.Rating {
		return false
	}
	return true
}
