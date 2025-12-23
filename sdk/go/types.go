package regstrava

// Error represents an API error
type Error struct {
	Message string `json:"error"`
}

func (e *Error) Error() string {
	return e.Message
}

// RateLimitInfo contains rate limit information from response headers
type RateLimitInfo struct {
	DailyLimit   int
	DailyUsed    int
	MonthlyLimit int
	MonthlyUsed  int
}

// HashLevel represents the level of detail in an invoice hash
type HashLevel int

const (
	HashLevelBasic    HashLevel = 1 // invoice_number + issuer_tax_id
	HashLevelStandard HashLevel = 2 // + amount + currency
	HashLevelDated    HashLevel = 3 // + invoice_date
	HashLevelFull     HashLevel = 4 // + buyer_tax_id
)

// String returns the string representation of the hash level
func (l HashLevel) String() string {
	switch l {
	case HashLevelBasic:
		return "Basic (L1)"
	case HashLevelStandard:
		return "Standard (L2)"
	case HashLevelDated:
		return "Dated (L3)"
	case HashLevelFull:
		return "Full (L4)"
	default:
		return "Unknown"
	}
}
