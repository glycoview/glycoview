package query

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/better-monitoring/glycoview/internal/model"
	"github.com/better-monitoring/glycoview/internal/store"
)

var bracePattern = regexp.MustCompile(`\{([^{}]+)\}`)

func ParseV1(values url.Values, defaultDateField string) store.Query {
	query := store.DefaultQuery()
	if defaultDateField == "" {
		defaultDateField = "date"
	}
	if count, err := strconv.Atoi(values.Get("count")); err == nil && count > 0 {
		query.Limit = count
	}
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		if !strings.HasPrefix(key, "find[") {
			continue
		}
		field, op := parseBracketFilter(key)
		if field == "" {
			continue
		}
		query.Filters = append(query.Filters, store.Filter{Field: field, Op: op, Value: vals[0]})
	}
	if rawFind := values.Get("find"); rawFind != "" {
		query.Filters = append(query.Filters, parseMongoFindJSON(rawFind)...)
	}
	if !hasIDFilter(query.Filters) && !hasDateFilter(query.Filters) {
		query.Filters = append(query.Filters, store.Filter{
			Field: defaultDateField,
			Op:    "gte",
			Value: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339),
		})
	}
	return query
}

func ParseV3(values url.Values) (store.Query, error) {
	query := store.DefaultQuery()
	query.SortField = "date"
	query.IncludeDeleted = false
	if values.Has("sort") && values.Has("sort$desc") {
		return query, fmt.Errorf("Parameters sort and sort_desc cannot be combined")
	}
	if limitRaw := values.Get("limit"); limitRaw != "" {
		limit, err := strconv.Atoi(limitRaw)
		if err != nil || limit <= 0 {
			return query, fmt.Errorf("Parameter limit out of tolerance")
		}
		query.Limit = limit
	}
	if skipRaw := values.Get("skip"); skipRaw != "" {
		skip, err := strconv.Atoi(skipRaw)
		if err != nil || skip < 0 {
			return query, fmt.Errorf("Parameter skip out of tolerance")
		}
		query.Skip = skip
	}
	if sortField := values.Get("sort"); sortField != "" {
		query.SortField = sortField
		query.SortDesc = false
	}
	if sortField := values.Get("sort$desc"); sortField != "" {
		query.SortField = sortField
		query.SortDesc = true
	}
	if fields := values.Get("fields"); fields != "" && fields != "_all" {
		for _, field := range strings.Split(fields, ",") {
			field = strings.TrimSpace(field)
			if field != "" {
				query.Fields = append(query.Fields, field)
			}
		}
	}
	for key, vals := range values {
		if len(vals) == 0 || isReservedV3Key(key) {
			continue
		}
		field, op := parseV3Filter(key)
		if field == "" {
			continue
		}
		query.Filters = append(query.Filters, store.Filter{Field: field, Op: op, Value: vals[0]})
	}
	return query, nil
}

func EchoV1(collection string, values url.Values) map[string]any {
	query := ParseV1(values, "date")
	return map[string]any{
		"storage": collection,
		"query":   query,
		"input": map[string]any{
			"find": values,
		},
	}
}

func Slice(records []model.Record, field, fieldType, prefix string, limit int) []model.Record {
	if prefix == "" {
		return records
	}
	patterns := ExpandBraces(prefix)
	filtered := make([]model.Record, 0, len(records))
	for _, record := range records {
		value := fmt.Sprint(model.PathValue(record.ToMap(true), field))
		if fieldType != "" && fmt.Sprint(model.PathValue(record.ToMap(true), "type")) != fieldType {
			continue
		}
		if matchesAnyPrefix(value, patterns) {
			filtered = append(filtered, record)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := model.PathValue(filtered[i].ToMap(true), "date")
		right := model.PathValue(filtered[j].ToMap(true), "date")
		return store.CompareValues(left, right, true)
	})
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered
}

func Times(records []model.Record, prefix, expr string, limit int) ([]model.Record, []string) {
	prefixes := ExpandBraces(prefix)
	expressions := ExpandBraces(expr)
	patterns := make([]string, 0, len(prefixes)*max(1, len(expressions)))
	if len(expressions) == 0 {
		expressions = []string{""}
	}
	for _, p := range prefixes {
		for _, e := range expressions {
			patterns = append(patterns, p+e)
		}
	}
	filtered := make([]model.Record, 0, len(records))
	for _, record := range records {
		dateString := fmt.Sprint(model.PathValue(record.ToMap(true), "dateString"))
		if dateString == "" {
			continue
		}
		if matchesAnyPrefix(dateString, patterns) {
			filtered = append(filtered, record)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := model.PathValue(filtered[i].ToMap(true), "date")
		right := model.PathValue(filtered[j].ToMap(true), "date")
		return store.CompareValues(left, right, true)
	})
	if limit > 0 && limit < len(filtered) {
		filtered = filtered[:limit]
	}
	return filtered, patterns
}

func ExpandBraces(input string) []string {
	if input == "" {
		return nil
	}
	if !strings.Contains(input, "{") {
		return []string{input}
	}
	match := bracePattern.FindStringSubmatchIndex(input)
	if match == nil {
		return []string{input}
	}
	body := input[match[2]:match[3]]
	parts := expandBody(body)
	prefix := input[:match[0]]
	suffix := input[match[1]:]
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		for _, expanded := range ExpandBraces(suffix) {
			out = append(out, prefix+part+expanded)
		}
	}
	return out
}

func parseBracketFilter(key string) (field, op string) {
	key = strings.TrimPrefix(key, "find[")
	parts := strings.Split(key, "][")
	if len(parts) == 0 {
		return "", ""
	}
	field = strings.TrimSuffix(parts[0], "]")
	op = "eq"
	if len(parts) > 1 {
		op = strings.TrimPrefix(strings.TrimSuffix(parts[1], "]"), "$")
	}
	return field, op
}

func parseV3Filter(key string) (field, op string) {
	if strings.Contains(key, "$") {
		parts := strings.SplitN(key, "$", 2)
		return parts[0], parts[1]
	}
	if strings.Contains(key, "_") {
		parts := strings.SplitN(key, "_", 2)
		return parts[0], parts[1]
	}
	return key, "eq"
}

func parseMongoFindJSON(raw string) []store.Filter {
	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil
	}
	var filters []store.Filter
	for field, value := range doc {
		if nested, ok := value.(map[string]any); ok {
			for op, subValue := range nested {
				filters = append(filters, store.Filter{Field: field, Op: strings.TrimPrefix(op, "$"), Value: fmt.Sprint(subValue)})
			}
			continue
		}
		filters = append(filters, store.Filter{Field: field, Op: "eq", Value: fmt.Sprint(value)})
	}
	return filters
}

func isReservedV3Key(key string) bool {
	switch key {
	case "limit", "skip", "sort", "sort$desc", "fields":
		return true
	default:
		return false
	}
}

func hasDateFilter(filters []store.Filter) bool {
	for _, filter := range filters {
		if filter.Field == "date" || filter.Field == "dateString" || filter.Field == "created_at" {
			return true
		}
	}
	return false
}

func hasIDFilter(filters []store.Filter) bool {
	for _, filter := range filters {
		if filter.Field == "_id" || filter.Field == "identifier" {
			return true
		}
	}
	return false
}

func expandBody(body string) []string {
	if strings.Contains(body, "..") {
		parts := strings.SplitN(body, "..", 2)
		start := parts[0]
		end := parts[1]
		startInt, err1 := strconv.Atoi(start)
		endInt, err2 := strconv.Atoi(end)
		if err1 == nil && err2 == nil {
			width := len(start)
			out := make([]string, 0, endInt-startInt+1)
			for i := startInt; i <= endInt; i++ {
				out = append(out, fmt.Sprintf("%0*d", width, i))
			}
			return out
		}
	}
	return strings.Split(body, ",")
}

func matchesAnyPrefix(value string, patterns []string) bool {
	for _, pattern := range patterns {
		regex := prefixPatternToRegexp(pattern)
		if regex.MatchString(value) {
			return true
		}
	}
	return false
}

func prefixPatternToRegexp(pattern string) *regexp.Regexp {
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\.\*`, ".*")
	regex := "^" + quoted
	return regexp.MustCompile(regex)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
