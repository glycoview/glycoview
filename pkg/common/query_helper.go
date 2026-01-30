package common

import (
	"fmt"
	"regexp"
)

type IQueryHelper interface {
	ParseFind(find string) (*QuerySpec, error)
	ParseFindWithIdOrTypeFiler(spec string, find string) (*QuerySpec, error)
}
type QueryHelper struct{}

func NewQueryHelper() *QueryHelper {
	return &QueryHelper{}
}

var (
	findOpRe = regexp.MustCompile(`^find\[([^\]]+)\]\[(\$\w+)\]$`)
	findEqRe = regexp.MustCompile(`^find\[([^\]]+)\]$`)
)

func (s *QueryHelper) ParseFindWithIdOrTypeFiler(spec string, find string) (*QuerySpec, error) {
	filter_spec, err := s.ParseFind(find)
	if err != nil {
		return nil, err
	}
	filter_spec.Filters = append(filter_spec.Filters, Filter{
		Field: "type",
		Op:    OpEq,
		Value: spec,
	})
	filter_spec.Filters = append(filter_spec.Filters, Filter{
		Field: "_id",
		Op:    OpEq,
		Value: spec,
	})

	return filter_spec, nil
}

func (s *QueryHelper) ParseFind(find string) (*QuerySpec, error) {
	spec := &QuerySpec{}

	matches := findOpRe.FindStringSubmatch(find)
	if len(matches) == 3 {
		field := matches[1]
		opStr := matches[2]
		op, err := mongoToRepoOp(opStr)
		if err != nil {
			return nil, err
		}
		spec.Filters = append(spec.Filters, Filter{
			Field: field,
			Op:    op,
			Value: find,
		})
		return spec, nil
	}

	matches = findEqRe.FindStringSubmatch(find)
	if len(matches) == 2 {
		field := matches[1]
		spec.Filters = append(spec.Filters, Filter{
			Field: field,
			Op:    OpEq,
			Value: find,
		})
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
