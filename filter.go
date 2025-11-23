package parcours

// FilterOp represents a filter operation type.
type FilterOp int

const (
	// Logical operators
	And FilterOp = iota
	Or
	Not

	// Comparison operators
	Eq       // ==
	Ne       // !=
	Gt       // >
	Gte      // >=
	Lt       // <
	Lte      // <=
	Contains // substring match
	Match    // regex match
)

// Filter represents a composable filter for log queries.
// Filters can be simple comparisons or complex logical combinations.
type Filter struct {
	Op       FilterOp  // Operation type
	Field    string    // Field name for comparison (empty for logical ops)
	Value    any       // Comparison value (nil for logical ops)
	Children []*Filter // Child filters for logical ops
}

// Sort represents a sort directive for log queries.
type Sort struct {
	Field string // Field name to sort by
	Desc  bool   // Sort descending if true, ascending if false
}

/*
// Helper constructors for common filter operations

// Eq creates an equality filter: field == value
func Eq(field string, value any) *Filter {
	return &Filter{
		Op:    Eq,
		Field: field,
		Value: value,
	}
}

// Ne creates a not-equal filter: field != value
func Ne(field string, value any) *Filter {
	return &Filter{
		Op:    Ne,
		Field: field,
		Value: value,
	}
}

// Gt creates a greater-than filter: field > value
func Gt(field string, value any) *Filter {
	return &Filter{
		Op:    Gt,
		Field: field,
		Value: value,
	}
}

// Gte creates a greater-than-or-equal filter: field >= value
func Gte(field string, value any) *Filter {
	return &Filter{
		Op:    Gte,
		Field: field,
		Value: value,
	}
}

// Lt creates a less-than filter: field < value
func Lt(field string, value any) *Filter {
	return &Filter{
		Op:    Lt,
		Field: field,
		Value: value,
	}
}

// Lte creates a less-than-or-equal filter: field <= value
func Lte(field string, value any) *Filter {
	return &Filter{
		Op:    Lte,
		Field: field,
		Value: value,
	}
}

// Contains creates a substring match filter
func Contains(field string, substring string) *Filter {
	return &Filter{
		Op:    Contains,
		Field: field,
		Value: substring,
	}
}

// Match creates a regex match filter
func Match(field string, pattern string) *Filter {
	return &Filter{
		Op:    Match,
		Field: field,
		Value: pattern,
	}
}

// And combines multiple filters with AND logic
func And(filters ...*Filter) *Filter {
	return &Filter{
		Op:       And,
		Children: filters,
	}
}

// Or combines multiple filters with OR logic
func Or(filters ...*Filter) *Filter {
	return &Filter{
		Op:       Or,
		Children: filters,
	}
}

// Not negates a filter
func Not(filter *Filter) *Filter {
	return &Filter{
		Op:       Not,
		Children: []*Filter{filter},
	}
}
*/
