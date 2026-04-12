package model

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Collection  string
	ID          string
	Data        map[string]any
	SrvCreated  int64
	SrvModified int64
	Subject     string
	IsValid     bool
	DeletedAt   *int64
}

func (r Record) Clone() Record {
	clone := r
	clone.Data = CloneMap(r.Data)
	return clone
}

func (r Record) Identifier() string {
	if id, ok := StringField(r.Data, "identifier"); ok && id != "" {
		return id
	}
	if id, ok := StringField(r.Data, "_id"); ok && id != "" {
		return id
	}
	return r.ID
}

func (r Record) WithData(data map[string]any) Record {
	clone := r.Clone()
	clone.Data = CloneMap(data)
	return clone
}

func (r Record) ToMap(includeMeta bool) map[string]any {
	out := CloneMap(r.Data)
	id := r.Identifier()
	if id != "" {
		out["_id"] = id
		out["identifier"] = id
	}
	if includeMeta {
		out["srvCreated"] = r.SrvCreated
		out["srvModified"] = r.SrvModified
		out["subject"] = r.Subject
		out["isValid"] = r.IsValid
		if r.DeletedAt != nil {
			out["deletedAt"] = *r.DeletedAt
		}
	}
	return out
}

func CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = CloneMap(typed)
		case []any:
			copied := make([]any, len(typed))
			for i, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					copied[i] = CloneMap(nested)
				} else {
					copied[i] = item
				}
			}
			out[k] = copied
		default:
			out[k] = typed
		}
	}
	return out
}

func Merge(dst, src map[string]any) map[string]any {
	out := CloneMap(dst)
	maps.Copy(out, src)
	return out
}

func StringField(data map[string]any, key string) (string, bool) {
	val, ok := data[key]
	if !ok || val == nil {
		return "", false
	}
	switch typed := val.(type) {
	case string:
		return typed, true
	case json.Number:
		return typed.String(), true
	case fmt.Stringer:
		return typed.String(), true
	default:
		return fmt.Sprint(typed), true
	}
}

func Int64Field(data map[string]any, key string) (int64, bool) {
	val, ok := data[key]
	if !ok || val == nil {
		return 0, false
	}
	switch typed := val.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		num, err := typed.Int64()
		return num, err == nil
	case string:
		if typed == "" {
			return 0, false
		}
		num, err := strconv.ParseInt(typed, 10, 64)
		return num, err == nil
	default:
		return 0, false
	}
}

func BoolField(data map[string]any, key string) (bool, bool) {
	val, ok := data[key]
	if !ok || val == nil {
		return false, false
	}
	switch typed := val.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(typed)
		return parsed, err == nil
	default:
		return false, false
	}
}

func ToUTCString(value string) (string, int, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		layouts := []string{
			"2006-01-02T15:04:05.000-0700",
			"2006-01-02T15:04:05-0700",
			"2006-01-02T15:04:05.000Z07:00",
			"2006-01-02T15:04:05Z07:00",
		}
		for _, layout := range layouts {
			parsed, err = time.Parse(layout, value)
			if err == nil {
				break
			}
		}
	}
	if err != nil {
		return "", 0, err
	}
	_, offsetSeconds := parsed.Zone()
	return parsed.UTC().Format("2006-01-02T15:04:05.000Z"), offsetSeconds / 60, nil
}

func NormalizeCollection(collection string) string {
	clean := strings.ToLower(strings.TrimSpace(collection))
	switch clean {
	case "device_status":
		return "devicestatus"
	case "profiles":
		return "profile"
	case "foods":
		return "food"
	default:
		return clean
	}
}

func PathValue(data map[string]any, path string) any {
	if data == nil || path == "" {
		return nil
	}
	if value, ok := data[path]; ok {
		return value
	}
	parts := splitPath(path)
	var current any = data
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = object[part]
		if !ok {
			return nil
		}
	}
	return current
}

func splitPath(path string) []string {
	if strings.Contains(path, ".") {
		return strings.Split(path, ".")
	}
	brackets := regexp.MustCompile(`\[([^\]]+)\]`)
	if brackets.MatchString(path) {
		path = brackets.ReplaceAllString(path, `.$1`)
		path = strings.TrimPrefix(path, ".")
		return strings.Split(path, ".")
	}
	return []string{path}
}
