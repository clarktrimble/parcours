package entity

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
	Op       FilterOp // Operation type
	Field    string   // Field name for comparison (empty for logical ops)
	Value    any      // Comparison value (nil for logical ops)
	Enabled  bool     // Whether this filter is active
	Children []Filter // Child filters for logical ops
}

// Sort represents a sort directive for log queries.
type Sort struct {
	Field string // Field name to sort by
	Desc  bool   // Sort descending if true, ascending if false
}
