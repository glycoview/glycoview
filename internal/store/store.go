package store

import (
	nsstore "github.com/glycoview/nightscout-api/store"

	"github.com/glycoview/glycoview/internal/model"
)

var (
	ErrNotFound = nsstore.ErrNotFound
	ErrGone     = nsstore.ErrGone
)

type Filter = nsstore.Filter
type Query = nsstore.Query
type Store = nsstore.Store

func NormalizeData(collection string, data map[string]any) (map[string]any, error) {
	clean, err := nsstore.NormalizeData(collection, data)
	if err != nil {
		return nil, err
	}
	collection = model.NormalizeCollection(collection)
	if collection == "entries" {
		if value, ok := data["sgv"].(string); ok && value != "" {
			clean["sgv"] = value
		}
		if value, ok := data["mbg"].(string); ok && value != "" {
			clean["mbg"] = value
		}
	}
	return clean, nil
}

func CalculateIdentifier(data map[string]any) string {
	return nsstore.CalculateIdentifier(data)
}

func DefaultQuery() Query {
	return nsstore.DefaultQuery()
}

func ApplyQuery(records []model.Record, query Query) []model.Record {
	return nsstore.ApplyQuery(records, query)
}

func SelectFields(record model.Record, fields []string) map[string]any {
	return nsstore.SelectFields(record, fields)
}

func GenerateIdentifier() string {
	return nsstore.GenerateIdentifier()
}

func DedupeKey(collection string, data map[string]any) string {
	return nsstore.DedupeKey(collection, data)
}

func CompareValues(left, right any, desc bool) bool {
	return nsstore.CompareValues(left, right, desc)
}
