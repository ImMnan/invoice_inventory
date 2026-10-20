package get

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/immnan/invoice_invoice/pkg"
)

func transactionFixtures() []pkg.Transactions {
	return []pkg.Transactions{
		{Id: "opening", Credit: 10000000, ClosingBalance: 10000000},
		{
			Id: "sale", InvoiceIds: []string{"INV-1", "INV-2"},
			Date: "2026-10-20", Narration: "Payment", Title: "Sale, \"goods\"\nreceived",
			Mode: "UPI", Credit: 5000000, ClosingBalance: 15000000,
			Label: []string{"sales", " CO180G-SRS ", ""}, For: []string{"Ankit", "Madhavbhai"},
		},
		{Id: "expense", InvoiceIds: []string{"INV-3"}, Label: []string{"Sales", "expense"}, For: []string{"Other"}},
	}
}

func TestTransactionFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter TransactionFilter
		want   []string
	}{
		{"all", TransactionFilter{}, []string{"opening", "sale", "expense"}},
		{"second invoice", TransactionFilter{InvoiceID: "INV-2"}, []string{"sale"}},
		{"exact invoice", TransactionFilter{InvoiceID: "INV"}, nil},
		{"label case and whitespace", TransactionFilter{Label: []string{" SALES "}}, []string{"sale", "expense"}},
		{"second label", TransactionFilter{Label: []string{"co180g-srs"}}, []string{"sale"}},
		{"exact label", TransactionFilter{Label: []string{"sale"}}, nil},
		{"all labels", TransactionFilter{Label: []string{"sales", "co180g-srs"}}, []string{"sale"}},
		{"missing one label", TransactionFilter{Label: []string{"sales", "missing"}}, nil},
		{"labels on separate transactions", TransactionFilter{Label: []string{"expense", "co180g-srs"}}, nil},
		{"duplicate labels", TransactionFilter{Label: []string{"sales", " SALES "}}, []string{"sale", "expense"}},
		{"second recipient", TransactionFilter{For: " MADHAVBHAI "}, []string{"sale"}},
		{"exact recipient", TransactionFilter{For: "An"}, nil},
		{"combined", TransactionFilter{InvoiceID: "INV-2", Label: []string{"SALES", "co180g-srs"}, For: "ankit"}, []string{"sale"}},
		{"combined mismatch", TransactionFilter{Label: []string{"sales"}, For: "Nobody"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, txn := range transactionFixtures() {
				if tt.filter.matches(txn) {
					got = append(got, txn.Id)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("matched %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransactionMultipleLabelFlags(t *testing.T) {
	transactions := []pkg.Transactions{
		{Id: "both", InvoiceIds: []string{"INV"}, Label: []string{"ops", "fuel", "extra"}},
		{Id: "ops-only", InvoiceIds: []string{"INV"}, Label: []string{"ops"}},
		{Id: "fuel-only", InvoiceIds: []string{"INV"}, Label: []string{"fuel"}},
		{Id: "neither", InvoiceIds: []string{"INV"}},
	}
	for _, args := range [][]string{
		{"-l", "ops", "-l", "fuel"},
		{"-l", "ops,fuel"},
		{"--label", " OPS ", "--label", "FUEL"},
		{"-l", "ops", "--label", "fuel,extra"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			cmd := TransactionCommand()
			if err := cmd.ParseFlags(args); err != nil {
				t.Fatal(err)
			}
			labels, err := cmd.Flags().GetStringSlice("label")
			if err != nil {
				t.Fatal(err)
			}
			for _, asCSV := range []bool{false, true} {
				var output bytes.Buffer
				options := transactionOptions{Filter: TransactionFilter{Label: labels}, CSV: asCSV}
				if err := writeTransactions(&output, transactions, options); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(output.String(), "both") {
					t.Fatalf("matching transaction excluded: %s", output.String())
				}
				for _, excluded := range []string{"ops-only", "fuel-only", "neither"} {
					if strings.Contains(output.String(), excluded) {
						t.Fatalf("transaction %q included: %s", excluded, output.String())
					}
				}
			}
		})
	}
}
func TestTransactionCSV(t *testing.T) {
	var output bytes.Buffer
	if err := writeTransactions(&output, transactionFixtures(), transactionOptions{CSV: true, Output: "wide"}); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&output).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 || len(rows[0]) != 10 {
		t.Fatalf("unexpected CSV shape: %v", rows)
	}
	want := []string{"sale", "2026-10-20", "Payment", "Sale, \"goods\"\nreceived", "UPI", "0", "50000", "150000", "Ankit, Madhavbhai", "INV-1, INV-2"}
	if !reflect.DeepEqual(rows[2], want) {
		t.Fatalf("CSV row = %q, want %q", rows[2], want)
	}
	if rows[1][1] != "" || rows[1][9] != "" {
		t.Fatalf("empty invoice and recipient arrays rendered incorrectly: %v", rows[1])
	}
}

func TestTransactionDecimalOutput(t *testing.T) {
	for _, narrationKey := range []string{"narration", "Narration"} {
		input := `[{"id":"bank","date":"2026-10-01","` + narrationKey + `":"UPI-PAYMENT","debit":0,"credit":10.25,"closingBalance":110.25}]`
		var transactions []pkg.Transactions
		if err := json.Unmarshal([]byte(input), &transactions); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if err := writeTransactions(&output, transactions, transactionOptions{CSV: true, Output: "wide"}); err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(&output).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if rows[1][2] != "UPI-PAYMENT" || rows[1][5] != "0" || rows[1][6] != "10.25" || rows[1][7] != "110.25" {
			t.Fatalf("incorrect decimal output: %q", rows[1])
		}
	}
}

func TestTransactionNarrationTruncation(t *testing.T) {
	for _, narration := range []string{"", "Short payment", strings.Repeat("a", 25), strings.Repeat("b", 26), strings.Repeat("\u0a95", 26)} {
		t.Run(narration, func(t *testing.T) {
			want := []rune(narration)
			if len(want) > 25 {
				want = want[:25]
			}
			transactions := []pkg.Transactions{{Id: "txn", Narration: narration}}
			for _, asCSV := range []bool{false, true} {
				var output bytes.Buffer
				if err := writeTransactions(&output, transactions, transactionOptions{CSV: asCSV}); err != nil {
					t.Fatal(err)
				}
				if asCSV {
					rows, err := csv.NewReader(&output).ReadAll()
					if err != nil {
						t.Fatal(err)
					}
					if rows[1][2] != string(want) {
						t.Fatalf("narration = %q, want %q", rows[1][2], string(want))
					}
				} else if string(want) != "" && !strings.Contains(output.String(), string(want)) {
					t.Fatalf("table missing narration %q: %s", string(want), output.String())
				}
				if len([]rune(narration)) > 25 && strings.Contains(output.String(), narration) {
					t.Fatal("output contains untruncated narration")
				}
				if transactions[0].Narration != narration {
					t.Fatal("stored narration was modified")
				}
			}
		})
	}
}

func TestTransactionWideOutput(t *testing.T) {
	narration := "UPI-" + strings.Repeat("payment", 8)
	transactions := []pkg.Transactions{{Id: "unique-transaction-id", Narration: narration, Label: []string{"sales"}}}
	for _, asCSV := range []bool{false, true} {
		for _, wide := range []bool{false, true} {
			for _, labels := range []bool{false, true} {
				options := transactionOptions{CSV: asCSV, ShowLabels: labels}
				if wide {
					options.Output = "wide"
				}
				var output bytes.Buffer
				if err := writeTransactions(&output, transactions, options); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(output.String(), transactions[0].Id) ||
					strings.Contains(output.String(), narration) != wide {
					t.Fatalf("csv=%v wide=%v: incorrect ID or narration output: %s", asCSV, wide, output.String())
				}
				if asCSV {
					rows, err := csv.NewReader(&output).ReadAll()
					if err != nil {
						t.Fatal(err)
					}
					columns, narrationColumn := 10, 2
					if rows[0][0] != "ID" || rows[1][0] != transactions[0].Id {
						t.Fatal("output missing ID column")
					}
					if labels {
						columns++
					}
					if len(rows[0]) != columns || len(rows[1]) != columns {
						t.Fatalf("unexpected CSV shape: %v", rows)
					}
					want := narration
					if !wide {
						want = string([]rune(narration)[:25])
					}
					if rows[1][narrationColumn] != want {
						t.Fatalf("narration = %q, want %q", rows[1][narrationColumn], want)
					}
				}
			}
		}
	}
	var output bytes.Buffer
	if err := writeTransactions(&output, transactions, transactionOptions{Output: "invalid"}); err == nil || output.Len() != 0 {
		t.Fatal("invalid output mode must fail without writing output")
	}
}

func TestTransactionLabels(t *testing.T) {
	tests := []struct {
		name   string
		filter TransactionFilter
		want   []string
	}{
		{"all", TransactionFilter{}, []string{"", "sales, CO180G-SRS", "Sales, expense", ""}},
		{"recipient", TransactionFilter{For: "Ankit"}, []string{"sales, CO180G-SRS"}},
		{"invoice", TransactionFilter{InvoiceID: "INV-3"}, []string{"Sales, expense"}},
		{"label", TransactionFilter{Label: []string{"expense"}}, []string{"Sales, expense"}},
		{"no labels", TransactionFilter{InvoiceID: "EMPTY"}, []string{""}},
		{"no match", TransactionFilter{Label: []string{"missing"}}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := append(transactionFixtures(), pkg.Transactions{Id: "no-labels", InvoiceIds: []string{"EMPTY"}})
			for i := range transactions {
				transactions[i].Title = strings.ReplaceAll(transactions[i].Title, "\n", " ")
			}
			for _, asCSV := range []bool{false, true} {
				var output bytes.Buffer
				options := transactionOptions{Filter: tt.filter, CSV: asCSV, ShowLabels: true}
				if err := writeTransactions(&output, transactions, options); err != nil {
					t.Fatal(err)
				}
				var baseline bytes.Buffer
				options.ShowLabels = false
				if err := writeTransactions(&baseline, transactions, options); err != nil {
					t.Fatal(err)
				}
				if !asCSV {
					lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
					if len(lines) != len(tt.want)+2 || !strings.HasSuffix(lines[0], "LABEL") {
						t.Fatalf("unexpected table shape: %q", output.String())
					}
					for i, label := range tt.want {
						if !strings.HasSuffix(strings.TrimRight(lines[i+1], " "), label) {
							t.Fatalf("row %d missing label cell %q: %q", i, label, lines[i+1])
						}
					}
					continue
				}
				rows, err := csv.NewReader(&output).ReadAll()
				if err != nil {
					t.Fatal(err)
				}
				transactionRows, err := csv.NewReader(&baseline).ReadAll()
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != len(transactionRows) || len(rows) != len(tt.want)+2 {
					t.Fatalf("unexpected CSV row count: %v", rows)
				}
				for i, row := range rows {
					label := "LABEL"
					if i == len(rows)-1 {
						label = ""
					} else if i > 0 {
						label = tt.want[i-1]
					}
					want := append(transactionRows[i], label)
					if !reflect.DeepEqual(row, want) {
						t.Fatalf("row = %q, want %q", row, want)
					}
				}
			}
		})
	}
}

func TestTransactionTotals(t *testing.T) {
	tests := []struct {
		name       string
		debit      pkg.Money
		credit     pkg.Money
		filter     TransactionFilter
		wantDebit  string
		wantCredit string
		percentage string
	}{
		{"positive", 10000, 12500, TransactionFilter{}, "100", "125", "25%"},
		{"negative", 10000, 9000, TransactionFilter{}, "100", "90", "-10%"},
		{"equal", 10000, 10000, TransactionFilter{}, "100", "100", "0%"},
		{"fractional", 300, 400, TransactionFilter{}, "3", "4", "33.33%"},
		{"decimal totals", 1025, 1230, TransactionFilter{}, "10.25", "12.30", "20%"},
		{"zero debit", 0, 10000, TransactionFilter{}, "0", "100", "N/A"},
		{"zero credit", 10000, 0, TransactionFilter{}, "100", "0", "-100%"},
		{"both zero", 0, 0, TransactionFilter{}, "0", "0", "0%"},
		{"invoice", 10000, 12500, TransactionFilter{InvoiceID: "INV"}, "40", "50", "25%"},
		{"label", 10000, 12500, TransactionFilter{Label: []string{"sales"}}, "40", "50", "25%"},
		{"recipient", 10000, 12500, TransactionFilter{For: "Ankit"}, "40", "50", "25%"},
		{"combined", 10000, 12500, TransactionFilter{InvoiceID: "INV", Label: []string{"sales"}, For: "Ankit"}, "40", "50", "25%"},
		{"no matches", 10000, 12500, TransactionFilter{InvoiceID: "missing"}, "0", "0", "0%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := []pkg.Transactions{
				{Id: "matched", Debit: tt.debit * 2 / 5, Credit: tt.credit * 2 / 5, InvoiceIds: []string{"INV"}, Label: []string{"sales"}, For: []string{"Ankit"}},
				{Id: "other", Debit: tt.debit - tt.debit*2/5, Credit: tt.credit - tt.credit*2/5},
			}
			for _, asCSV := range []bool{false, true} {
				for _, labels := range []bool{false, true} {
					var output bytes.Buffer
					options := transactionOptions{CSV: asCSV, ShowLabels: labels, Filter: tt.filter}
					if err := writeTransactions(&output, transactions, options); err != nil {
						t.Fatal(err)
					}
					if asCSV {
						rows, err := csv.NewReader(&output).ReadAll()
						if err != nil {
							t.Fatal(err)
						}
						want := make([]string, len(rows[0]))
						want[0], want[5], want[6], want[7] = "TOTAL", tt.wantDebit, tt.wantCredit, tt.percentage
						if !reflect.DeepEqual(rows[len(rows)-1], want) {
							t.Fatalf("total = %q, want %q", rows[len(rows)-1], want)
						}
					} else {
						lines := strings.Split(strings.TrimSpace(output.String()), "\n")
						want := []string{"TOTAL", tt.wantDebit, tt.wantCredit, tt.percentage}
						if got := strings.Fields(lines[len(lines)-1]); !reflect.DeepEqual(got, want) {
							t.Fatalf("total = %q, want %q", got, want)
						}
					}
				}
			}
		})
	}
}

func TestTransactionTable(t *testing.T) {
	var output bytes.Buffer
	options := transactionOptions{Filter: TransactionFilter{InvoiceID: "INV-2"}, Output: "wide"}
	if err := writeTransactions(&output, transactionFixtures(), options); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ID", "INVOICE", "sale", "INV-1, INV-2", "Ankit, Madhavbhai", "150000"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("output missing %q: %s", want, output.String())
		}
	}
	if strings.Contains(output.String(), "opening") || strings.Contains(output.String(), "expense") {
		t.Fatalf("nonmatching transactions included: %s", output.String())
	}
}

type failingTransactionWriter struct{}

func (failingTransactionWriter) Write([]byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestTransactionOutputErrors(t *testing.T) {
	for _, asCSV := range []bool{false, true} {
		err := writeTransactions(failingTransactionWriter{}, transactionFixtures(), transactionOptions{CSV: asCSV})
		if !errors.Is(err, io.ErrClosedPipe) {
			t.Fatalf("csv=%v: error = %v, want closed pipe", asCSV, err)
		}
	}
}

func TestTransactionCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	data, err := json.Marshal(transactionFixtures())
	if err != nil {
		t.Fatal(err)
	}
	transactionPath := filepath.Join(dir, "transactions.json")
	if err := os.WriteFile(transactionPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	config := "data:\n  transactionData: " + strconv.Quote(transactionPath) + "\n"
	config += "dataOct:\n  transactionData: " + strconv.Quote(transactionPath) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lvsConfig.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		args    []string
		want    string
		wantErr string
	}{
		{args: []string{"INV-2", "-l", "SALES", "-f", "ankit", "--csv", "-m", "10"}, want: "INV-1, INV-2"},
		{args: []string{"INV-2", "-l", "sales", "-l", "co180g-srs", "--csv"}, want: "INV-1, INV-2"},
		{args: []string{"INV-2", "-l", "sales,co180g-srs", "--csv"}, want: "INV-1, INV-2"},
		{args: []string{"-o", "wide", "--csv"}, want: "ID,DATE,NARRATION"},
		{args: []string{"--output", "wide"}, want: "ID"},
		{args: []string{"-o", "invalid"}, wantErr: "unsupported output"},
		{args: []string{"--label", "sales", "--for", "Ankit", "--show-labels", "--csv"}, want: "\"Ankit, Madhavbhai\",\"INV-1, INV-2\",\"sales, CO180G-SRS\"\n"},
		{args: []string{"--month", "-1"}, wantErr: "month must be"},
		{args: []string{"--month", "14"}, wantErr: "month must be"},
		{args: []string{"--month", "1"}, wantErr: "transactionData is not configured"},
		{args: []string{"INV-1", "INV-2"}, wantErr: "accepts at most 1 arg"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			cmd := TransactionCommand()
			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), tt.want) {
				t.Fatalf("output = %q, want %q", output.String(), tt.want)
			}
		})
	}
	if err := os.WriteFile(transactionPath, []byte("{invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := printTransactions(io.Discard, transactionOptions{}); err == nil {
		t.Fatal("invalid JSON should return an error")
	}
	if err := os.Remove(transactionPath); err != nil {
		t.Fatal(err)
	}
	if err := printTransactions(io.Discard, transactionOptions{}); err == nil {
		t.Fatal("missing transaction file should return an error")
	}
}
