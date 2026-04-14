package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/better-monitoring/glycoview/internal/model"
)

var (
	ErrNotFound = errors.New("record not found")
	ErrGone     = errors.New("record deleted")
)

var nightscoutIdentifierNamespace = uuid.UUID{'N', 'i', 'g', 'h', 't', 's', 'c', 'o', 'u', 't', 'R', 'o', 'c', 'k', 's', '!'}

type Filter struct {
	Field string
	Op    string
	Value string
}

type Query struct {
	Filters        []Filter
	Limit          int
	Skip           int
	SortField      string
	SortDesc       bool
	Fields         []string
	IncludeDeleted bool
}

type Store interface {
	Create(ctx context.Context, collection string, data map[string]any, subject string) (model.Record, bool, error)
	Get(ctx context.Context, collection, identifier string) (model.Record, error)
	Search(ctx context.Context, collection string, query Query) ([]model.Record, error)
	Replace(ctx context.Context, collection, identifier string, data map[string]any, subject string) (model.Record, bool, error)
	Patch(ctx context.Context, collection, identifier string, patch map[string]any, subject string) (model.Record, error)
	Delete(ctx context.Context, collection, identifier string, permanent bool, subject string) error
	DeleteMatching(ctx context.Context, collection string, query Query, permanent bool, subject string) (int, error)
	History(ctx context.Context, collection string, since int64, limit int) ([]model.Record, error)
	LastModified(ctx context.Context, collections []string) (map[string]int64, error)
}

func NormalizeData(collection string, data map[string]any) (map[string]any, error) {
	clean := model.CloneMap(data)
	collection = model.NormalizeCollection(collection)

	if rawDate, exists := clean["date"]; exists && rawDate != nil {
		millis, offset, err := parseDateValue(rawDate)
		if err != nil {
			return nil, fmt.Errorf("bad or missing date field")
		}
		clean["date"] = millis
		if _, exists := clean["utcOffset"]; !exists && offset != nil {
			clean["utcOffset"] = *offset
		}
	}
	if value, ok := model.StringField(clean, "created_at"); ok && value != "" {
		normalized, offset, err := model.ToUTCString(value)
		if err != nil {
			return nil, fmt.Errorf("bad created_at field")
		}
		clean["created_at"] = normalized
		if _, exists := clean["utcOffset"]; !exists {
			clean["utcOffset"] = offset
		}
	}
	if value, ok := model.Int64Field(clean, "date"); ok {
		if value <= 100000000000 {
			return nil, fmt.Errorf("bad or missing date field")
		}
	} else if value, ok := model.StringField(clean, "created_at"); ok && value != "" {
		parsed, _ := time.Parse("2006-01-02T15:04:05.000Z", value)
		if !parsed.IsZero() {
			clean["date"] = parsed.UnixMilli()
		}
	}
	if clean["date"] == nil {
		if collection == "entries" || collection == "treatments" || collection == "devicestatus" || collection == "food" {
			return nil, fmt.Errorf("bad or missing date field")
		}
	} else if _, ok := model.StringField(clean, "created_at"); !ok && collection != "entries" && collection != "settings" {
		if date, ok := model.Int64Field(clean, "date"); ok {
			clean["created_at"] = time.UnixMilli(date).UTC().Format("2006-01-02T15:04:05.000Z")
		}
	}
	if offset, exists := clean["utcOffset"]; exists && offset != nil {
		num, err := toInt64(offset)
		if err != nil || num < -1440 || num > 1440 {
			return nil, fmt.Errorf("bad or missing utcOffset field")
		}
		clean["utcOffset"] = int(num)
	}
	if notes, ok := model.StringField(clean, "notes"); ok {
		clean["notes"] = sanitizeNotes(notes)
	}
	if _, ok := model.StringField(clean, "type"); !ok && collection == "entries" {
		clean["type"] = "sgv"
	}
	if _, ok := model.StringField(clean, "identifier"); !ok {
		if existingID, ok := model.StringField(clean, "_id"); ok && existingID != "" {
			clean["identifier"] = existingID
		}
	}
	return clean, nil
}

func CalculateIdentifier(data map[string]any) string {
	device, ok := model.StringField(data, "device")
	if !ok || strings.TrimSpace(device) == "" {
		return ""
	}
	date, ok := model.Int64Field(data, "date")
	if !ok || date <= 0 {
		return ""
	}
	key := fmt.Sprintf("%s_%d", device, date)
	if eventType, ok := model.StringField(data, "eventType"); ok && strings.TrimSpace(eventType) != "" {
		key += "_" + eventType
	}
	return uuid.NewSHA1(nightscoutIdentifierNamespace, []byte(key)).String()
}

func DefaultQuery() Query {
	return Query{
		Limit:          10,
		SortField:      "date",
		SortDesc:       true,
		IncludeDeleted: false,
	}
}

func ApplyQuery(records []model.Record, query Query) []model.Record {
	filtered := make([]model.Record, 0, len(records))
	for _, record := range records {
		if !query.IncludeDeleted && !record.IsValid {
			continue
		}
		if matchesAll(record, query.Filters) {
			filtered = append(filtered, record.Clone())
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		left := filtered[i].ToMap(true)
		right := filtered[j].ToMap(true)
		return CompareValues(left[query.SortField], right[query.SortField], query.SortDesc)
	})
	start := min(query.Skip, len(filtered))
	filtered = filtered[start:]
	if query.Limit > 0 && query.Limit < len(filtered) {
		filtered = filtered[:query.Limit]
	}
	return filtered
}

func SelectFields(record model.Record, fields []string) map[string]any {
	full := record.ToMap(true)
	if len(fields) == 0 {
		return full
	}
	out := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := full[field]; ok {
			out[field] = value
		}
	}
	if len(fields) > 0 {
		delete(out, "_id")
	}
	return out
}

func GenerateIdentifier() string {
	return uuid.NewString()
}

func DedupeKey(collection string, data map[string]any) string {
	collection = model.NormalizeCollection(collection)
	switch collection {
	case "entries":
		date, _ := model.Int64Field(data, "date")
		kind, _ := model.StringField(data, "type")
		return fmt.Sprintf("%s:%d:%s", collection, date, kind)
	case "treatments":
		createdAt, _ := model.StringField(data, "created_at")
		eventType, _ := model.StringField(data, "eventType")
		return fmt.Sprintf("%s:%s:%s", collection, createdAt, eventType)
	case "devicestatus":
		createdAt, _ := model.StringField(data, "created_at")
		device, _ := model.StringField(data, "device")
		return fmt.Sprintf("%s:%s:%s", collection, createdAt, device)
	case "food", "profile":
		createdAt, _ := model.StringField(data, "created_at")
		return fmt.Sprintf("%s:%s", collection, createdAt)
	default:
		if identifier, ok := model.StringField(data, "identifier"); ok && identifier != "" {
			return fmt.Sprintf("%s:%s", collection, identifier)
		}
		return ""
	}
}

func matchesAll(record model.Record, filters []Filter) bool {
	full := record.ToMap(true)
	for _, filter := range filters {
		if !matchFilter(model.PathValue(full, filter.Field), filter) {
			return false
		}
	}
	return true
}

func matchFilter(value any, filter Filter) bool {
	switch filter.Field {
	case "dateString":
		if millis, ok := toInt64(value); ok == nil {
			value = time.UnixMilli(millis).UTC().Format("2006-01-02T15:04:05")
		}
	}
	if filter.Op == "exists" {
		want := strings.EqualFold(filter.Value, "true")
		return (value != nil) == want
	}
	if filter.Op == "re" || isRegexLiteral(filter.Value) {
		pattern, regexErr := compileRegexFilter(filter.Value)
		if regexErr == nil {
			matched := pattern.MatchString(strings.TrimSpace(fmt.Sprint(value)))
			if filter.Op == "ne" || filter.Op == "nin" {
				return !matched
			}
			return matched
		}
	}
	leftNum, leftNumErr := toFloat64(value)
	rightNum, rightNumErr := strconv.ParseFloat(filter.Value, 64)
	if leftNumErr == nil && rightNumErr == nil {
		switch filter.Op {
		case "eq":
			return leftNum == rightNum
		case "ne":
			return leftNum != rightNum
		case "gte":
			return leftNum >= rightNum
		case "lte":
			return leftNum <= rightNum
		case "gt":
			return leftNum > rightNum
		case "lt":
			return leftNum < rightNum
		case "in":
			return inNumericSet(leftNum, filter.Value)
		case "nin":
			return !inNumericSet(leftNum, filter.Value)
		}
	}
	leftTime, leftTimeErr := toTime(value)
	rightTime, rightTimeErr := toTime(filter.Value)
	if leftTimeErr == nil && rightTimeErr == nil {
		switch filter.Op {
		case "eq":
			return leftTime.Equal(rightTime)
		case "ne":
			return !leftTime.Equal(rightTime)
		case "gte":
			return leftTime.After(rightTime) || leftTime.Equal(rightTime)
		case "lte":
			return leftTime.Before(rightTime) || leftTime.Equal(rightTime)
		case "gt":
			return leftTime.After(rightTime)
		case "lt":
			return leftTime.Before(rightTime)
		}
	}
	left := strings.TrimSpace(fmt.Sprint(value))
	right := strings.TrimSpace(filter.Value)
	switch filter.Op {
	case "eq":
		return left == right
	case "ne":
		return left != right
	case "gte":
		return left >= right
	case "lte":
		return left <= right
	case "gt":
		return left > right
	case "lt":
		return left < right
	case "in":
		return inStringSet(left, right)
	case "nin":
		return !inStringSet(left, right)
	default:
		return false
	}
}

func CompareValues(left, right any, desc bool) bool {
	if leftNum, err := toFloat64(left); err == nil {
		if rightNum, err := toFloat64(right); err == nil {
			if desc {
				return leftNum > rightNum
			}
			return leftNum < rightNum
		}
	}
	if leftTime, err := toTime(left); err == nil {
		if rightTime, err := toTime(right); err == nil {
			if desc {
				return leftTime.After(rightTime)
			}
			return leftTime.Before(rightTime)
		}
	}
	leftString := fmt.Sprint(left)
	rightString := fmt.Sprint(right)
	if desc {
		return leftString > rightString
	}
	return leftString < rightString
}

func toFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case float32:
		return float64(typed), nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(typed, 64)
	default:
		return 0, fmt.Errorf("not numeric")
	}
}

func toInt64(value any) (int64, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil
	case int:
		return int64(typed), nil
	case int32:
		return int64(typed), nil
	case float64:
		return int64(typed), nil
	case float32:
		return int64(typed), nil
	case json.Number:
		return typed.Int64()
	case string:
		return strconv.ParseInt(typed, 10, 64)
	default:
		return 0, fmt.Errorf("not int")
	}
}

func toTime(value any) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case int64:
		return time.UnixMilli(typed), nil
	case int:
		return time.UnixMilli(int64(typed)), nil
	case float64:
		return time.UnixMilli(int64(typed)), nil
	case string:
		layouts := []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05.000Z",
			"2006-01-02",
			"2006-01",
		}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, typed)
			if err == nil {
				return parsed, nil
			}
		}
		if millis, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return time.UnixMilli(millis), nil
		}
		return time.Time{}, fmt.Errorf("not time")
	default:
		return time.Time{}, fmt.Errorf("not time")
	}
}

func isRegexLiteral(value string) bool {
	return strings.HasPrefix(value, "/") && strings.Count(value, "/") >= 2
}

func compileRegexFilter(value string) (*regexp.Regexp, error) {
	if isRegexLiteral(value) {
		last := strings.LastIndex(value, "/")
		body := value[1:last]
		flags := value[last+1:]
		if strings.Contains(flags, "i") {
			body = "(?i)" + body
		}
		return regexp.Compile(body)
	}
	return regexp.Compile(value)
}

func inStringSet(value, raw string) bool {
	for _, part := range strings.Split(raw, "|") {
		if value == part {
			return true
		}
	}
	return false
}

func inNumericSet(value float64, raw string) bool {
	for _, part := range strings.Split(raw, "|") {
		if parsed, err := strconv.ParseFloat(part, 64); err == nil && parsed == value {
			return true
		}
	}
	return false
}

func sanitizeNotes(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if strings.Contains(trimmed, "<img") {
		return "<img>"
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseDateValue(value any) (int64, *int, error) {
	switch typed := value.(type) {
	case int64:
		return typed, nil, nil
	case int:
		return int64(typed), nil, nil
	case int32:
		return int64(typed), nil, nil
	case float64:
		return int64(typed), nil, nil
	case float32:
		return int64(typed), nil, nil
	case json.Number:
		if asInt, err := typed.Int64(); err == nil {
			return asInt, nil, nil
		}
		asFloat, err := typed.Float64()
		if err != nil {
			return 0, nil, err
		}
		return int64(asFloat), nil, nil
	case string:
		raw := strings.TrimSpace(typed)
		if raw == "" {
			return 0, nil, fmt.Errorf("empty date")
		}
		if millis, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return millis, nil, nil
		}
		layouts := []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05,000Z07:00",
			"2006-01-02T15:04:05,000-0700",
			"2006-01-02T15:04:05.000-0700",
			"2006-01-02T15:04:05-0700",
			"2006-01-02T15:04:05.000Z07:00",
			"2006-01-02T15:04:05Z07:00",
		}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, raw)
			if err == nil {
				_, offsetSeconds := parsed.Zone()
				offset := offsetSeconds / 60
				return parsed.UnixMilli(), &offset, nil
			}
		}
		return 0, nil, fmt.Errorf("bad date")
	default:
		return 0, nil, fmt.Errorf("bad date")
	}
}
