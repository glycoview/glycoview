package common

import (
	"fmt"
	"net/url"
	"regexp"
)

type IQueryHelper interface {
	ParseFind(values url.Values) (*QuerySpec, error)
}
type QueryHelper struct{}

func NewQueryHelper() *QueryHelper {
	return &QueryHelper{}
}

var (
	findOpRe = regexp.MustCompile(`^find\[([^\]]+)\]\[(\$\w+)\]$`)
	findEqRe = regexp.MustCompile(`^find\[([^\]]+)\]$`)
)

func (s *QueryHelper) ParseFind(values url.Values) (*QuerySpec, error) {
	spec := &QuerySpec{}

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}

		val := vals[0]

		if m := findOpRe.FindStringSubmatch(key); len(m) == 3 {
			op, err := mongoToRepoOp(m[2])
			if err != nil {
				return nil, err
			}
			spec.Filters = append(spec.Filters, Filter{
				Field: m[1],
				Op:    op,
				Value: val,
			})
			continue
		}

		if m := findEqRe.FindStringSubmatch(key); len(m) == 2 {
			spec.Filters = append(spec.Filters, Filter{
				Field: m[1],
				Op:    OpEq,
				Value: val,
			})
		}
	}

	return spec, nil
}

func mongoToRepoOp(op string) (Operator, error) {
	switch op {
	case "$eq":
		return OpEq, nil
	case "$ne":
		return OpNe, nil
	case "$gt":
		return OpGt, nil
	case "$gte":
		return OpGte, nil
	case "$lt":
		return OpLt, nil
	case "$lte":
		return OpLte, nil
	case "$in":
		return OpIn, nil
	default:
		return "", fmt.Errorf("unsupported find operator: %s", op)
	}
}
