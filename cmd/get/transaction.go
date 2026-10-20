package get

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/pkg"
	"github.com/spf13/cobra"
)

type TransactionFilter struct {
	InvoiceID string
	Label     []string
	For       string
}

type transactionOptions struct {
	Filter     TransactionFilter
	Month      int
	CSV        bool
	ShowLabels bool
	Output     string
}

var transactionCmd = TransactionCommand()

func TransactionCommand() *cobra.Command {
	var options transactionOptions
	cmd := &cobra.Command{
		Use:     "transaction [invoice_id]",
		Short:   "Get transactions for the organization",
		Aliases: []string{"transactions", "txn"},
		Long:    "Fetch transactions for the configured month, optionally filtered by invoice, label, and recipient.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Filter.InvoiceID = ""
			if len(args) > 0 {
				options.Filter.InvoiceID = args[0]
			}
			return printTransactions(cmd.OutOrStdout(), options)
		},
	}
	cmd.Flags().BoolVar(&options.CSV, "csv", false, "Output in CSV format")
	cmd.Flags().StringVarP(&options.Output, "output", "o", "", "Use wide to show complete narration")
	cmd.Flags().StringSliceVarP(&options.Filter.Label, "label", "l", nil, "Require all labels (comma-separated or repeated)")
	cmd.Flags().StringVarP(&options.Filter.For, "for", "f", "", "Filter transactions by 'for' field")
	cmd.Flags().BoolVar(&options.ShowLabels, "show-labels", false, "Show a label column for matching transactions")
	cmd.Flags().IntVarP(&options.Month, "month", "m", 0, "Month to fetch transactions for (default is current month)")
	return cmd
}

func init() {
	GetCmd.AddCommand(transactionCmd)
}

func printTransactions(output io.Writer, options transactionOptions) error {
	if options.Month < 0 || options.Month > 13 {
		return fmt.Errorf("month must be between 0 and 13 (0: current, 1-12: calendar month, 13: test)")
	}
	_, _, _, _, transactionDB, err := ConfigData(options.Month)
	if err != nil {
		return fmt.Errorf("fetching config data: %w", err)
	}
	if transactionDB == "" {
		return fmt.Errorf("transactionData is not configured for month %d", options.Month)
	}
	db := &pkg.JsLocalDB{TransactionFile: transactionDB}
	data, err := db.Transactions()
	if err != nil {
		return fmt.Errorf("fetching transaction data: %w", err)
	}

	var transactions []pkg.Transactions
	if err := json.Unmarshal(data, &transactions); err != nil {
		return fmt.Errorf("unmarshalling transaction data: %w", err)
	}
	return writeTransactions(output, transactions, options)
}

func writeTransactions(output io.Writer, transactions []pkg.Transactions, options transactionOptions) error {
	if options.Output != "" && options.Output != "wide" {
		return fmt.Errorf("unsupported output %q: use -o wide or omit --output", options.Output)
	}
	wide := options.Output == "wide"
	rows := [][]string{{"ID", "DATE", "NARRATION", "TITLE", "MODE", "DEBIT", "CREDIT", "BALANCE", "FOR", "INVOICE"}}
	if options.ShowLabels {
		rows[0] = append(rows[0], "LABEL")
	}
	var totalDebit, totalCredit pkg.Money
	for _, txn := range transactions {
		if !options.Filter.matches(txn) {
			continue
		}
		totalDebit += txn.Debit
		totalCredit += txn.Credit
		narration := []rune(txn.Narration)
		if !wide && len(narration) > 25 {
			narration = narration[:25]
		}
		row := []string{
			txn.Id, txn.Date, string(narration),
			txn.Title, txn.Mode, txn.Debit.String(), txn.Credit.String(),
			txn.ClosingBalance.String(), strings.Join(txn.For, ", "),
			strings.Join(txn.InvoiceIds, ", "),
		}
		if options.ShowLabels {
			var labels []string
			for _, label := range txn.Label {
				if label = strings.TrimSpace(label); label != "" {
					labels = append(labels, label)
				}
			}
			row = append(row, strings.Join(labels, ", "))
		}
		rows = append(rows, row)
	}
	percentage := "0%"
	if totalDebit != 0 {
		difference := (float64(totalCredit) - float64(totalDebit)) / float64(totalDebit) * 100
		value := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", difference), "0"), ".")
		if value == "-0" {
			value = "0"
		}
		percentage = value + "%"
	} else if totalCredit != 0 {
		percentage = "N/A"
	}
	total := make([]string, len(rows[0]))
	total[0] = "TOTAL"
	total[5], total[6], total[7] = totalDebit.String(), totalCredit.String(), percentage
	rows = append(rows, total)
	if options.CSV {
		writer := csv.NewWriter(output)
		if err := writer.WriteAll(rows); err != nil {
			return fmt.Errorf("writing transaction CSV: %w", err)
		}
		return nil
	}
	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		if _, err := fmt.Fprintln(writer, strings.Join(row, "\t")); err != nil {
			return fmt.Errorf("writing transaction output: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("writing transaction output: %w", err)
	}
	return nil
}
