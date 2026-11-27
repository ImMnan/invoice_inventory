package apply

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/immnan/invoice_invoice/pkg"
)

func TestApplyTransactionCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	current := filepath.Join(dir, "current.json")
	october := filepath.Join(dir, "october.json")
	csvPath := filepath.Join(dir, "arbitrary-name.csv")
	statement := "\n Date , Narration , Value Dat , Debit Amount , Credit Amount , Chq/Ref Number , Closing Balance \n" +
		"01/10/26,UPI-PAYMENT,01/10/26,0,10.25,00000000123,110.25\n"
	if err := os.WriteFile(csvPath, []byte(statement), 0600); err != nil {
		t.Fatal(err)
	}
	config := "data:\n  transactionData: " + strconv.Quote(current) + "\ndataOct:\n  transactionData: " + strconv.Quote(october) + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lvsConfig.yaml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		t.Helper()
		cmd := ApplyCommand()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs(append([]string{"-f", csvPath, "--hdfc"}, args...))
		err := cmd.Execute()
		return output.String(), err
	}
	output, err := run()
	if err != nil || !strings.Contains(output, "Preview only") || !strings.Contains(output, current) {
		t.Fatalf("preview = %s, %v", output, err)
	}
	if _, err := os.Stat(current); !os.IsNotExist(err) {
		t.Fatal("preview created destination")
	}
	output, err = run("--approve")
	if err != nil || !strings.Contains(output, "Applied 1 new") {
		t.Fatalf("apply = %s, %v", output, err)
	}
	before, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	var transactions []pkg.Transactions
	if err := json.Unmarshal(before, &transactions); err != nil || len(transactions) != 1 || transactions[0].Credit.String() != "10.25" {
		t.Fatalf("incorrect saved data: %s, %v", before, err)
	}
	output, err = run("--approve", "-m", "0")
	if err != nil || !strings.Contains(output, "Applied 0 new") {
		t.Fatalf("repeat = %s, %v", output, err)
	}
	after, _ := os.ReadFile(current)
	if !bytes.Equal(before, after) {
		t.Fatal("repeat import changed file")
	}
	output, err = run("--csv", "-m", "10")
	if err != nil {
		t.Fatal(err)
	}
	previewRows, err := csv.NewReader(strings.NewReader(output)).ReadAll()
	if err != nil || len(previewRows) != 2 || previewRows[1][5] != "10.25" {
		t.Fatalf("CSV preview: %s, %v", output, err)
	}
	if _, err := run("--approve", "-m", "10"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(october); err != nil {
		t.Fatal("explicit -m did not select monthly destination:", err)
	}
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"-m", "-1"}, "month must be"},
		{[]string{"-m", "14"}, "month must be"},
		{[]string{"-m", "9"}, "transactionData is not configured"},
		{[]string{"--color", "red"}, "only supported for stock"},
		{[]string{"--print", "plain"}, "only supported for stock"},
	} {
		_, err := run(test.args...)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%v: error %v, expected %s", test.args, err, test.want)
		}
	}
	if err := os.WriteFile(csvPath, []byte(strings.Replace(statement, "10.25", "bad", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := run("--approve"); err == nil {
		t.Fatal("malformed statement should fail")
	}
	after, _ = os.ReadFile(current)
	if !bytes.Equal(before, after) {
		t.Fatal("malformed statement changed file")
	}
}

func TestHDFCHeaderValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		csv  string
		fail bool
	}{
		{"bank", "Date,Narration,Debit Amount,Credit Amount,Chq/Ref Number,Closing Balance\n01/10/26,UPI-ONE,0,10,00123,100\n", false},
		{"bank BOM", "\ufeffDate,Narration,Debit Amount,Credit Amount,Chq/Ref Number,Closing Balance\n01/10/26,UPI-ONE,0,10,00123,100\n", false},
		{"bank reordered", "Narration,Closing Balance,Date,Chq/Ref Number,Credit Amount,Debit Amount\nUPI-ONE,100,01/10/26,00123,10,0\n", false},
		{"stock", "Type,Invoice,Product_Id,Date\n", true},
		{"partial bank", "Date,Narration,Closing Balance\n", true},
		{"duplicate", "Type,Invoice,Product_Id,Type\n", true},
		{"unknown", "Date,Something\n", true},
		{"empty", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var statement pkg.TransactionStatement = pkg.TransactionHDFC{}
			_, err := statement.Parse(strings.NewReader(test.csv))
			if (err != nil) != test.fail {
				t.Fatalf("unexpected parse error: %v", err)
			}
		})
	}
}

func TestHDFCFlagRequiredForTransactions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	destination := filepath.Join(dir, "transactions.json")
	if err := os.WriteFile("lvsConfig.yaml", []byte("data:\n  transactionData: "+strconv.Quote(destination)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("bank.csv", []byte("Date,Narration,Debit Amount,Credit Amount,Chq/Ref Number,Closing Balance\n01/10/26,UPI-ONE,0,10,00123,100\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-f", "bank.csv", "--approve"}, {"-f", "bank.csv", "--hdfc=false", "--approve"}} {
		cmd := ApplyCommand()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatal("bank CSV was automatically imported without --hdfc")
		}
	}
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"--hdfc"}, "--file is required"},
		{[]string{"-f", "missing.csv", "--hdfc"}, "opening transaction CSV"},
	} {
		cmd := ApplyCommand()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(test.args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("%v: got %v, want %s", test.args, err, test.want)
		}
	}
}

type failingStatement struct {
	err error
}

func (s failingStatement) Parse(io.Reader) ([]pkg.BankTransaction, error) {
	return nil, s.err
}

func TestApplyTransactionsUsesStatementInterface(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)
	if err := os.WriteFile("lvsConfig.yaml", []byte("data:\n  transactionData: transactions.json\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("statement.csv", nil, 0600); err != nil {
		t.Fatal(err)
	}
	parseErr := errors.New("institution-specific parse failure")
	err := applyTransactions(io.Discard, "statement.csv", true, false, 0, failingStatement{err: parseErr})
	if !errors.Is(err, parseErr) {
		t.Fatalf("parser error not propagated: %v", err)
	}
	if _, err := os.Stat("transactions.json"); !os.IsNotExist(err) {
		t.Fatal("failed parser created transaction data")
	}
}
