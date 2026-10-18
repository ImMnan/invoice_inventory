package apply

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/immnan/invoice_invoice/cmd/get"
	"github.com/immnan/invoice_invoice/pkg"
)

func applyTransactions(output io.Writer, path string, approve, asCSV bool, month int, statement pkg.TransactionStatement) error {
	_, _, _, _, destination, err := get.ConfigData(month)
	if err != nil {
		return fmt.Errorf("fetching transaction configuration: %w", err)
	}
	if destination == "" {
		return fmt.Errorf("transactionData is not configured for month %d", month)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening transaction CSV: %w", err)
	}
	defer file.Close()
	rows, err := statement.Parse(file)
	if err != nil {
		return err
	}
	plan, err := pkg.PlanTransactionImport(destination, rows)
	if err != nil {
		return err
	}
	if err := writeTransactionPreview(output, plan, asCSV); err != nil {
		return err
	}
	if approve {
		if err := plan.Save(); err != nil {
			return err
		}
		if !asCSV {
			_, err = fmt.Fprintf(output, "Applied %d new transactions; skipped %d existing transactions.\n", len(plan.New), plan.Skipped)
		}
	} else if !asCSV {
		_, err = fmt.Fprintln(output, "Preview only. Use --approve to save.")
	}
	return err
}

func writeTransactionPreview(output io.Writer, plan *pkg.TransactionImport, asCSV bool) error {
	rows := [][]string{{"ID", "DATE", "NARRATION", "MODE", "DEBIT", "CREDIT", "BALANCE"}}
	for _, txn := range plan.New {
		rows = append(rows, []string{txn.Id, txn.Date, txn.Narration, txn.Mode, txn.Debit.String(), txn.Credit.String(), txn.ClosingBalance.String()})
	}
	if asCSV {
		writer := csv.NewWriter(output)
		if err := writer.WriteAll(rows); err != nil {
			return fmt.Errorf("writing transaction preview CSV: %w", err)
		}
		return nil
	}
	if _, err := fmt.Fprintf(output, "Transaction destination: %s\nNew: %d; already imported: %d\n", plan.Path, len(plan.New), plan.Skipped); err != nil {
		return fmt.Errorf("writing transaction preview: %w", err)
	}
	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				if _, err := fmt.Fprint(writer, "\t"); err != nil {
					return fmt.Errorf("writing transaction preview: %w", err)
				}
			}
			if _, err := fmt.Fprint(writer, cell); err != nil {
				return fmt.Errorf("writing transaction preview: %w", err)
			}
		}
		if _, err := fmt.Fprintln(writer); err != nil {
			return fmt.Errorf("writing transaction preview: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("writing transaction preview: %w", err)
	}
	return nil
}
