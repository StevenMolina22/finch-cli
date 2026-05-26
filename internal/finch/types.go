package finch

const (
	TypeIncome  = "income"
	TypeExpense = "expense"
)

type Transaction struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Amount      string `json:"amount"`
	Category    string `json:"category"`
	Desc        string `json:"desc"`
	Date        string `json:"date"`
	AmountCents int64  `json:"-"`
}

type AddInput struct {
	Type        string
	AmountCents int64
	Category    string
	Desc        string
	Date        string
}

type ListFilter struct {
	Month    string
	Category string
}

type Summary struct {
	Month        string `json:"month,omitempty"`
	Income       string `json:"income"`
	Expense      string `json:"expense"`
	Net          string `json:"net"`
	IncomeCents  int64  `json:"-"`
	ExpenseCents int64  `json:"-"`
	NetCents     int64  `json:"-"`
}

func NewSummary(month string, incomeCents, expenseCents int64) Summary {
	netCents := incomeCents - expenseCents
	return Summary{
		Month:        month,
		Income:       FormatAmount(incomeCents),
		Expense:      FormatAmount(expenseCents),
		Net:          FormatAmount(netCents),
		IncomeCents:  incomeCents,
		ExpenseCents: expenseCents,
		NetCents:     netCents,
	}
}

func NewTransaction(id int64, typ string, amountCents int64, category, desc, date string) Transaction {
	return Transaction{
		ID:          id,
		Type:        typ,
		Amount:      FormatAmount(amountCents),
		Category:    category,
		Desc:        desc,
		Date:        date,
		AmountCents: amountCents,
	}
}
