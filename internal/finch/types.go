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
	Tags        string `json:"tags"`
	Recurring   string `json:"recurring"`
	AmountCents int64  `json:"-"`
}

type AddInput struct {
	Type        string
	AmountCents int64
	Category    string
	Desc        string
	Date        string
	Tags        string
	Recurring   string
}

type ListFilter struct {
	Month    string
	Category string
	Limit    int
}

type EditInput struct {
	ID          int64
	AmountCents *int64
	Category    *string
	Desc        *string
	Tags        *string
	Recurring   *string
}

type ExportFilter struct {
	Month string
}

type ImportRow struct {
	Type     string
	Amount   string
	Category string
	Desc     string
	Date     string
	Tags     string
	Recurring string
}

type TopCategory struct {
	Category string `json:"category"`
	Amount   string `json:"amount"`
}

type Summary struct {
	Month         string        `json:"month,omitempty"`
	Income        string        `json:"income"`
	Expense       string        `json:"expense"`
	Net           string        `json:"net"`
	TopCategories []TopCategory `json:"top_categories,omitempty"`
	IncomeCents   int64         `json:"-"`
	ExpenseCents  int64         `json:"-"`
	NetCents      int64         `json:"-"`
}

func NewSummary(month string, incomeCents, expenseCents int64, topCategories []TopCategory) Summary {
	netCents := incomeCents - expenseCents
	if topCategories == nil {
		topCategories = []TopCategory{}
	}
	return Summary{
		Month:         month,
		Income:        FormatAmount(incomeCents),
		Expense:       FormatAmount(expenseCents),
		Net:           FormatAmount(netCents),
		TopCategories: topCategories,
		IncomeCents:   incomeCents,
		ExpenseCents:  expenseCents,
		NetCents:      netCents,
	}
}

func NewTransaction(id int64, typ string, amountCents int64, category, desc, date, tags, recurring string) Transaction {
	return Transaction{
		ID:          id,
		Type:        typ,
		Amount:      FormatAmount(amountCents),
		Category:    category,
		Desc:        desc,
		Date:        date,
		Tags:        tags,
		Recurring:   recurring,
		AmountCents: amountCents,
	}
}
