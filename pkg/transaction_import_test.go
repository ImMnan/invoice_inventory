package pkg

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const bankHeader = "Date,Narration,Value Dat,Debit Amount,Credit Amount,Chq/Ref Number,Closing Balance\n"

const bankRows = "01/10/26,UPI-ONE,01/10/26,10.25,0,000000000000123,89.75\n" +
	"01/10/26,REV-UPI-ONE,01/10/26,0,10.25,000000000000123,100\n" +
	"02/10/26,CASH DEPOSIT BY - SELF,02/10/26,0,20,000000000000000,120\n" +
	"03/10/26,IB FUNDS TRANSFER,03/10/26,5.10,0,NET123,114.90\n"

func TestMoney(t *testing.T) {
	for _, input := range []string{"0", "145095.74", "-0.01", "100000", "1e2"} {
		var amount Money
		if err := json.Unmarshal([]byte(input), &amount); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(amount)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip Money
		if err := json.Unmarshal(encoded, &roundTrip); err != nil || roundTrip != amount {
			t.Fatalf("%s did not round trip: %s, %v", input, encoded, err)
		}
	}
	for _, input := range []string{"null", `"1"`, "0.001", "1e100", "NaN"} {
		var amount Money
		if err := json.Unmarshal([]byte(input), &amount); err == nil {
			t.Fatalf("accepted invalid amount %s", input)
		}
	}
}

func TestTransactionHDFCParse(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader("\n " + bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("got %d transactions", len(rows))
	}
	for i, mode := range []string{"UPI", "UPI", "CASH", "NETB"} {
		if rows[i].Mode != mode || rows[i].InvoiceIds == nil || rows[i].Label == nil || rows[i].For == nil {
			t.Fatalf("incorrect defaults: %+v", rows[i])
		}
	}
	if rows[0].Date != "2026-10-01" || rows[0].Debit.String() != "10.25" {
		t.Fatalf("incorrect date or money: %+v", rows[0])
	}
	for _, bad := range []string{
		strings.Replace(bankRows, "10.25", "bad", 1),
		strings.Replace(bankRows, "01/10/26", "31/02/26", 1),
		strings.Replace(bankRows, "89.75", "89.76", 1),
		strings.Replace(bankRows, "NET123,114.90", "NET123,115", 1),
		strings.Replace(bankRows, "000000000000123", "", 1),
		strings.Replace(bankRows, "10.25", "-10.25", 1),
		strings.Replace(bankRows, "03/10/26", "01/10/26", 1),
	} {
		if _, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bad)); err == nil {
			t.Fatalf("accepted invalid statement: %s", bad)
		}
	}
}

func TestTransactionImportAppendAndPreserve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transactions.json")
	initial := []byte(`[{"id":"opening","date":"2026-09-30","narration":"","credit":100,"debit":0,"closingBalance":100,"title":"Keep me","invoiceIds":["INV"],"label":["sales"],"for":["Person"],"notes":"unchanged","custom":{"value":42}}]`)
	if err := os.WriteFile(path, initial, 0640); err != nil {
		t.Fatal(err)
	}
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanTransactionImport(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.New) != 4 || plan.Skipped != 0 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	seen := map[string]bool{}
	for _, txn := range plan.New {
		if seen[txn.Id] || txn.Id == "000000000000000" {
			t.Fatalf("duplicate/zero ID %s", txn.Id)
		}
		seen[txn.Id] = true
	}
	if plan.New[0].Id != "000000000000123" || !strings.HasPrefix(plan.New[1].Id, "000000000000123-") || len(plan.New[2].Id) != 15 {
		t.Fatalf("incorrect generated IDs: %+v", plan.New)
	}
	unchanged, _ := os.ReadFile(path)
	if !bytes.Equal(initial, unchanged) {
		t.Fatal("planning modified the destination")
	}
	if err := plan.Save(); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records []json.RawMessage
	if err := json.Unmarshal(saved, &records); err != nil || len(records) != 5 {
		t.Fatalf("invalid saved data: %s, %v", saved, err)
	}
	var before, after any
	if err := json.Unmarshal(initial, &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(records[0], &after); err != nil {
		t.Fatal(err)
	}
	beforeRecord, _ := json.Marshal(before.([]any)[0])
	afterRecord, _ := json.Marshal(after)
	if !bytes.Equal(beforeRecord, afterRecord) {
		t.Fatalf("existing record changed: %s", records[0])
	}
	for _, key := range []string{`"narration"`, `"title": ""`, `"notes": ""`, `"invoiceIds": []`, `"label": []`, `"for": []`} {
		if !bytes.Contains(saved, []byte(key)) {
			t.Fatalf("missing JSON shape %s: %s", key, saved)
		}
	}
	repeat, err := PlanTransactionImport(path, rows)
	if err != nil || len(repeat.New) != 0 || repeat.Skipped != 4 {
		t.Fatalf("repeat import: %+v, %v", repeat, err)
	}
	if err := repeat.Save(); err != nil {
		t.Fatal(err)
	}
	afterRepeat, _ := os.ReadFile(path)
	if !bytes.Equal(saved, afterRepeat) {
		t.Fatal("no-op import rewrote file")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0640 {
		t.Fatalf("permissions changed: %v", info.Mode())
	}
}

func TestTransactionImportOverlap(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transactions.json")
	plan, err := PlanTransactionImport(path, rows[:2])
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Save(); err != nil {
		t.Fatal(err)
	}
	overlap, err := PlanTransactionImport(path, rows[1:])
	if err != nil || len(overlap.New) != 2 || overlap.Skipped != 1 {
		t.Fatalf("overlap import: %+v, %v", overlap, err)
	}
	if err := overlap.Save(); err != nil {
		t.Fatal(err)
	}
	for _, subset := range [][]BankTransaction{rows, rows[:2], rows[1:3], rows[2:]} {
		plan, err := PlanTransactionImport(path, subset)
		if err != nil || len(plan.New) != 0 {
			t.Fatalf("already imported subset: %+v, %v", plan, err)
		}
	}
	conflict := append([]BankTransaction(nil), rows...)
	conflict[1].Narration = "Changed"
	if _, err := PlanTransactionImport(path, conflict); err == nil {
		t.Fatal("conflicting overlap should fail")
	}
}

func TestTransactionImportRejectsUnsafeChanges(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	for _, existing := range []string{
		`{bad`,
		`null`,
		`[{"id":"x","closingBalance":99,"date":"2026-09-30"}]`,
		`[{"id":"x","closingBalance":100,"date":"2026-10-31"}]`,
		`[{"id":"x"},{"id":"x"}]`,
		`[null]`,
	} {
		path := filepath.Join(t.TempDir(), "transactions.json")
		if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := PlanTransactionImport(path, rows); err == nil {
			t.Fatalf("accepted unsafe existing data: %s", existing)
		}
		got, _ := os.ReadFile(path)
		if string(got) != existing {
			t.Fatal("failed import changed existing file")
		}
	}
	path := filepath.Join(t.TempDir(), "transactions.json")
	plan, err := PlanTransactionImport(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := plan.Save(); err == nil {
		t.Fatal("concurrent change should prevent save")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "[]" {
		t.Fatal("concurrent change was overwritten")
	}
}

func TestSuppliedBankStatement(t *testing.T) {
	file, err := os.Open("../oct-statement.csv")
	if os.IsNotExist(err) {
		t.Skip("optional local oct-statement.csv is not present")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	rows, err := (TransactionHDFC{}).Parse(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 74 {
		t.Fatalf("got %d rows, want 74", len(rows))
	}
	if rows[0].Reference != "0000464432327661" || rows[0].ClosingBalance.String() != "145095.74" ||
		rows[73].ClosingBalance.String() != "19211.92" {
		t.Fatalf("sample values changed: first=%+v, last=%+v", rows[0], rows[73])
	}
	plan, err := PlanTransactionImport(filepath.Join(t.TempDir(), "transactions.json"), rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := PlanTransactionImport(plan.Path, rows)
	if err != nil || len(again.New) != 0 {
		t.Fatalf("sample repeat failed: %+v, %v", again, err)
	}
}

func TestTransactionRepeatedIdenticalRows(t *testing.T) {
	statement := bankHeader + "01/10/26,BANK INFO,01/10/26,0,0,000000000000000,100\n" +
		"01/10/26,BANK INFO,01/10/26,0,0,000000000000000,100\n"
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(statement))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transactions.json")
	first, err := PlanTransactionImport(path, rows[:1])
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	extended, err := PlanTransactionImport(path, rows)
	if err != nil || len(extended.New) != 1 {
		t.Fatalf("repeated occurrence lost: %+v, %v", extended, err)
	}
	if extended.New[0].Id == first.New[0].Id {
		t.Fatal("identical occurrences received the same ID")
	}
	if err := extended.Save(); err != nil {
		t.Fatal(err)
	}
	again, err := PlanTransactionImport(path, rows)
	if err != nil || len(again.New) != 0 {
		t.Fatalf("repeat duplicated identical occurrences: %+v, %v", again, err)
	}
	if _, err := PlanTransactionImport(path, rows[:1]); err == nil {
		t.Fatal("ambiguous single-row overlap should fail")
	}
}

func TestTransactionImportNewOnlyAndLegacyRecords(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transactions.json")
	first, err := PlanTransactionImport(path, rows[:2])
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	legacy := bytes.ReplaceAll(data, []byte(`"narration"`), []byte(`"Narration"`))
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	repeated, err := PlanTransactionImport(path, rows[:2])
	if err != nil || len(repeated.New) != 0 {
		t.Fatalf("legacy Narration not recognized: %+v, %v", repeated, err)
	}
	next, err := PlanTransactionImport(path, rows[2:])
	if err != nil || len(next.New) != 2 {
		t.Fatalf("new-only statement rejected: %+v, %v", next, err)
	}
	if err := next.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if bytes.Count(data, []byte(`"Narration"`)) != 2 {
		t.Fatal("existing legacy record keys were rewritten")
	}
}

func TestTransactionImportLockAndCleanup(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "transactions.json")
	plan, err := PlanTransactionImport(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".lock", []byte("another importer"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := plan.Save(); err == nil || !strings.Contains(err.Error(), "locking") {
		t.Fatalf("existing lock did not prevent import: %v", err)
	}
	lock, _ := os.ReadFile(path + ".lock")
	if string(lock) != "another importer" {
		t.Fatal("another importer's lock was changed")
	}
	if err := os.Remove(path + ".lock"); err != nil {
		t.Fatal(err)
	}
	if err := plan.Save(); err != nil {
		t.Fatal(err)
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 1 || files[0].Name() != "transactions.json" {
		t.Fatalf("temporary files or locks left behind: %v, %v", files, err)
	}
}

func TestConcurrentTransactionImports(t *testing.T) {
	rows, err := (TransactionHDFC{}).Parse(strings.NewReader(bankHeader + bankRows))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "transactions.json")
	first, err := PlanTransactionImport(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanTransactionImport(path, rows)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, plan := range []*TransactionImport{first, second} {
		go func() {
			<-start
			results <- plan.Save()
		}()
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("got %d successful concurrent imports, want 1", successes)
	}
	data, _ := os.ReadFile(path)
	var saved []Transactions
	if err := json.Unmarshal(data, &saved); err != nil || len(saved) != len(rows) {
		t.Fatalf("concurrent imports damaged data: %s, %v", data, err)
	}
}
