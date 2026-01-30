package common

type Operator string

const (
	OpEq  Operator = "="
	OpNe  Operator = "!="
	OpGt  Operator = ">"
	OpGte Operator = ">="
	OpLt  Operator = "<"
	OpLte Operator = "<="
	OpIn  Operator = "IN"
)

type Filter struct {
	Field string
	Op    Operator
	Value any
}

type QuerySpec struct {
	Filters []Filter
	Limit   int
	Offset  int
	Sort    []Sort
}

type Sort struct {
	Field string
	Desc  bool
}
