package pkg

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type TransactionStatement interface {
	Parse(io.Reader) ([]BankTransaction, error)
}

type TransactionHDFC struct{}

type BankTransaction struct {
	Transactions
	Reference string
	Row       int
}

type TransactionImport struct {
	Path    string
	New     []Transactions
	Skipped int

	original []byte
	records  []json.RawMessage
	existed  bool
	mode     os.FileMode
}

func statementReader(input io.Reader) *csv.Reader {
	reader := csv.NewReader(input)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	return reader
}

func readStatementRecord(reader *csv.Reader) ([]string, error) {
	for {
		record, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}
		return record, nil
	}
}

func statementHeaders(reader *csv.Reader) (map[string]int, int, error) {
	record, err := readStatementRecord(reader)
	if err != nil {
		return nil, 0, fmt.Errorf("reading CSV header: %w", err)
	}
	headers := make(map[string]int, len(record))
	for i, column := range record {
		column = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(column), "\ufeff")))
		if _, exists := headers[column]; exists || column == "" {
			return nil, 0, fmt.Errorf("duplicate or empty CSV header %q", column)
		}
		headers[column] = i
	}
	return headers, len(record), nil
}

func (TransactionHDFC) Parse(input io.Reader) ([]BankTransaction, error) {
	reader := statementReader(input)
	headers, width, err := statementHeaders(reader)
	if err != nil {
		return nil, err
	}
	for _, column := range []string{"date", "narration", "debit amount", "credit amount", "chq/ref number", "closing balance"} {
		if _, ok := headers[column]; !ok {
			return nil, fmt.Errorf("missing bank statement column %q", column)
		}
	}
	var transactions []BankTransaction
	for {
		record, err := readStatementRecord(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading bank statement: %w", err)
		}
		line, _ := reader.FieldPos(0)
		if len(record) != width {
			return nil, fmt.Errorf("CSV row %d has %d fields, expected %d", line, len(record), width)
		}
		field := func(column string) string { return strings.TrimSpace(record[headers[column]]) }
		date, err := time.Parse("02/01/06", field("date"))
		if err != nil {
			return nil, fmt.Errorf("CSV row %d: invalid Date: %w", line, err)
		}
		txn := BankTransaction{
			Transactions: Transactions{
				Date: date.Format(time.DateOnly), Narration: field("narration"),
				InvoiceIds: []string{}, Label: []string{}, For: []string{},
			},
			Reference: field("chq/ref number"), Row: line,
		}
		if txn.Reference == "" || txn.Narration == "" {
			return nil, fmt.Errorf("CSV row %d: Narration and Chq/Ref Number must not be blank", line)
		}
		for _, amount := range []struct {
			column string
			target *Money
		}{
			{"debit amount", &txn.Debit}, {"credit amount", &txn.Credit}, {"closing balance", &txn.ClosingBalance},
		} {
			value := field(amount.column)
			if value == "" && amount.column != "closing balance" {
				value = "0"
			}
			*amount.target, err = ParseMoney(value)
			if err != nil {
				return nil, fmt.Errorf("CSV row %d, %s: %w", line, amount.column, err)
			}
		}
		if txn.Debit < 0 || txn.Credit < 0 || (txn.Debit > 0 && txn.Credit > 0) {
			return nil, fmt.Errorf("CSV row %d: debit and credit must be nonnegative and cannot both be positive", line)
		}
		narration := strings.ToUpper(txn.Narration)
		switch {
		case strings.HasPrefix(narration, "UPI-"), strings.HasPrefix(narration, "REV-UPI-"):
			txn.Mode = "UPI"
		case narration == "CASH", strings.HasPrefix(narration, "CASH "), strings.HasPrefix(narration, "CASH-"):
			txn.Mode = "CASH"
		default:
			txn.Mode = "NETB"
		}
		if len(transactions) > 0 {
			if err := checkTransactionContinuity(transactions[len(transactions)-1].Transactions, txn.Transactions); err != nil {
				return nil, fmt.Errorf("CSV row %d: %w", line, err)
			}
		}
		transactions = append(transactions, txn)
	}
	if len(transactions) == 0 {
		return nil, fmt.Errorf("bank statement contains no transactions")
	}
	return transactions, nil
}

func checkTransactionContinuity(previous, next Transactions) error {
	if _, err := time.Parse(time.DateOnly, previous.Date); err != nil {
		return fmt.Errorf("cannot verify continuity: existing transaction %q has invalid date %q", previous.Id, previous.Date)
	}
	if _, err := time.Parse(time.DateOnly, next.Date); err != nil {
		return fmt.Errorf("invalid transaction date %q", next.Date)
	}
	if next.Date < previous.Date {
		return fmt.Errorf("transactions are not in chronological order; cannot append older transactions")
	}
	expected := big.NewInt(int64(previous.ClosingBalance))
	expected.Sub(expected, big.NewInt(int64(next.Debit)))
	expected.Add(expected, big.NewInt(int64(next.Credit)))
	if expected.Cmp(big.NewInt(int64(next.ClosingBalance))) != 0 {
		return fmt.Errorf("closing balance does not continue from %s (next debit %s, credit %s, balance %s)",
			previous.ClosingBalance, next.Debit, next.Credit, next.ClosingBalance)
	}
	return nil
}

func transactionFingerprint(txn Transactions) string {
	fields := []string{txn.Date, strings.TrimSpace(txn.Narration), txn.Debit.String(), txn.Credit.String(), txn.ClosingBalance.String()}
	data, _ := json.Marshal(fields)
	return string(data)
}

func zeroReference(reference string) bool {
	return reference != "" && strings.Trim(reference, "0") == ""
}

func matchesBankTransaction(existing Transactions, incoming BankTransaction) bool {
	if transactionFingerprint(existing) != transactionFingerprint(incoming.Transactions) {
		return false
	}
	return zeroReference(incoming.Reference) || existing.Id == incoming.Reference ||
		strings.HasPrefix(existing.Id, incoming.Reference+"-")
}

func PlanTransactionImport(path string, incoming []BankTransaction) (*TransactionImport, error) {
	if len(incoming) == 0 {
		return nil, fmt.Errorf("no transactions to import")
	}
	plan := &TransactionImport{Path: path, mode: 0600}
	info, err := os.Lstat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("checking transaction destination: %w", err)
	}
	var existing []Transactions
	if err == nil {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("transaction destination must be a regular file, not a directory or symlink")
		}
		plan.existed, plan.mode = true, info.Mode().Perm()
		plan.original, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading existing transactions: %w", err)
		}
		if !bytes.HasPrefix(bytes.TrimSpace(plan.original), []byte("[")) {
			return nil, fmt.Errorf("existing transaction data must be a JSON array")
		}
		if err := json.Unmarshal(plan.original, &plan.records); err != nil {
			return nil, fmt.Errorf("reading existing transaction JSON: %w", err)
		}
		if err := json.Unmarshal(plan.original, &existing); err != nil {
			return nil, fmt.Errorf("reading existing transaction values: %w", err)
		}
		for i, record := range plan.records {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(record, &fields); err != nil {
				return nil, fmt.Errorf("reading existing transaction %d: %w", i+1, err)
			}
			for _, field := range []string{"debit", "credit", "closingBalance"} {
				if _, exists := fields[field]; !exists {
					return nil, fmt.Errorf("existing transaction %d is missing %q; cannot verify bank history", i+1, field)
				}
			}
		}
	}
	usedIDs := make(map[string]bool, len(existing))
	occurrences := make(map[string]int, len(existing))
	for _, txn := range existing {
		if strings.TrimSpace(txn.Id) == "" || usedIDs[txn.Id] {
			return nil, fmt.Errorf("existing transactions have a blank or duplicate ID %q", txn.Id)
		}
		usedIDs[txn.Id] = true
		occurrences[transactionFingerprint(txn)]++
	}

	start, candidates := 0, 0
	for i := range existing {
		if !matchesBankTransaction(existing[i], incoming[0]) {
			continue
		}
		j := 0
		for i+j < len(existing) && j < len(incoming) && matchesBankTransaction(existing[i+j], incoming[j]) {
			j++
		}
		if i+j == len(existing) || j == len(incoming) {
			if j > start {
				start, candidates = j, 1
			} else if j == start {
				candidates++
			}
		}
	}
	if candidates > 1 {
		return nil, fmt.Errorf("ambiguous transaction overlap; provide a statement with more surrounding rows")
	}
	if candidates == 0 && len(existing) > 0 {
		// An apparent overlap that does not form a contiguous sequence is unsafe.
		for _, row := range incoming {
			for _, txn := range existing {
				if transactionFingerprint(txn) == transactionFingerprint(row.Transactions) {
					return nil, fmt.Errorf("conflicting transaction overlap at CSV row %d; existing records were not changed", row.Row)
				}
			}
		}
		if err := checkTransactionContinuity(existing[len(existing)-1], incoming[0].Transactions); err != nil {
			return nil, fmt.Errorf("no matching overlap: %w; provide an overlapping statement", err)
		}
	}
	plan.Skipped = start
	incomingCounts := make(map[string]int, len(incoming))
	for _, row := range incoming {
		incomingCounts[row.Reference+"\x00"+transactionFingerprint(row.Transactions)]++
	}
	for _, row := range incoming[start:] {
		key := transactionFingerprint(row.Transactions)
		existingMatches := 0
		for _, txn := range existing {
			if matchesBankTransaction(txn, row) {
				existingMatches++
			}
		}
		if existingMatches > 0 && incomingCounts[row.Reference+"\x00"+key] <= existingMatches {
			return nil, fmt.Errorf("CSV row %d repeats an existing transaction outside the matched overlap; cannot safely append", row.Row)
		}
		occurrence := occurrences[key]
		occurrences[key]++
		txn := row.Transactions
		txn.Id = row.Reference
		hash := sha256.Sum256([]byte(row.Reference + "\x00" + key + "\x00" + strconv.Itoa(occurrence)))
		if zeroReference(row.Reference) {
			txn.Id = fmt.Sprintf("%015d", binary.BigEndian.Uint64(hash[:8])%999999999999999+1)
		} else if usedIDs[txn.Id] {
			txn.Id += fmt.Sprintf("-%x", hash[:8])
		}
		if usedIDs[txn.Id] {
			return nil, fmt.Errorf("generated transaction ID collision at CSV row %d: %q", row.Row, txn.Id)
		}
		usedIDs[txn.Id] = true
		plan.New = append(plan.New, txn)
	}
	return plan, nil
}

func (plan *TransactionImport) checkUnchanged() error {
	current, err := os.ReadFile(plan.Path)
	if !plan.existed && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("rechecking transaction destination: %w", err)
	}
	if !plan.existed || !bytes.Equal(current, plan.original) {
		return fmt.Errorf("transaction destination changed during import; retry with a fresh preview")
	}
	info, err := os.Lstat(plan.Path)
	if err != nil {
		return fmt.Errorf("rechecking transaction file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != plan.mode {
		return fmt.Errorf("transaction destination type or permissions changed during import")
	}
	return nil
}

func (plan *TransactionImport) Save() (err error) {
	if len(plan.New) == 0 {
		return plan.checkUnchanged()
	}
	lockPath := plan.Path + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("locking transaction destination (another import may be running; inspect %s): %w", lockPath, err)
	}
	defer func() {
		if removeErr := os.Remove(lockPath); removeErr != nil {
			err = errors.Join(err, fmt.Errorf("removing transaction import lock: %w", removeErr))
		}
	}()
	if err := lock.Close(); err != nil {
		return fmt.Errorf("closing transaction import lock: %w", err)
	}
	if err := plan.checkUnchanged(); err != nil {
		return err
	}
	records := append([]json.RawMessage(nil), plan.records...)
	for _, txn := range plan.New {
		data, err := json.Marshal(txn)
		if err != nil {
			return fmt.Errorf("encoding new transaction: %w", err)
		}
		records = append(records, data)
	}
	data, err := json.MarshalIndent(records, "", "    ")
	if err != nil {
		return fmt.Errorf("encoding transaction JSON: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(plan.Path), ".lvs-transactions-*")
	if err != nil {
		return fmt.Errorf("creating temporary transaction file: %w", err)
	}
	tempPath := file.Name()
	defer func() {
		if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, fmt.Errorf("cleaning temporary transaction file: %w", removeErr))
		}
	}()
	if _, err = file.Write(append(data, '\n')); err == nil {
		err = file.Chmod(plan.mode)
	}
	if err == nil {
		err = file.Sync()
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return fmt.Errorf("writing transaction file: %w", err)
	}
	if err := plan.checkUnchanged(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, plan.Path); err != nil {
		return fmt.Errorf("replacing transaction file: %w", err)
	}
	directory, err := os.Open(filepath.Dir(plan.Path))
	if err != nil {
		return fmt.Errorf("transactions were saved but opening their directory for sync failed: %w", err)
	}
	err = errors.Join(directory.Sync(), directory.Close())
	if err != nil {
		return fmt.Errorf("transactions were saved but syncing their directory failed: %w", err)
	}
	return err
}
