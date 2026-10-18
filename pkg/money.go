package pkg

import (
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// Money stores hundredths of a currency unit and encodes as a JSON number.
type Money int64

var moneyNumber = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?([eE][+-]?[0-9]{1,2})?$`)

func ParseMoney(value string) (Money, error) {
	value = strings.TrimSpace(value)
	if len(value) > 64 || !moneyNumber.MatchString(value) {
		return 0, fmt.Errorf("invalid monetary amount %q", value)
	}
	amount, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, fmt.Errorf("invalid monetary amount %q", value)
	}
	amount.Mul(amount, big.NewRat(100, 1))
	if !amount.IsInt() || !amount.Num().IsInt64() {
		return 0, fmt.Errorf("amount %q exceeds supported range or has fractional cents", value)
	}
	return Money(amount.Num().Int64()), nil
}

func (m Money) String() string {
	digits := strconv.FormatInt(int64(m), 10)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	for len(digits) < 3 {
		digits = "0" + digits
	}
	whole, fraction := digits[:len(digits)-2], digits[len(digits)-2:]
	if fraction == "00" {
		return sign + whole
	}
	return sign + whole + "." + fraction
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	if !json.Valid(data) {
		return fmt.Errorf("invalid monetary JSON %q", data)
	}
	amount, err := ParseMoney(string(data))
	if err != nil {
		return err
	}
	*m = amount
	return nil
}
