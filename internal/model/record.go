package model

import nsmodel "github.com/glycoview/nightscout-api/model"

type Record = nsmodel.Record

func CloneMap(in map[string]any) map[string]any {
	return nsmodel.CloneMap(in)
}

func Merge(dst, src map[string]any) map[string]any {
	return nsmodel.Merge(dst, src)
}

func StringField(data map[string]any, key string) (string, bool) {
	return nsmodel.StringField(data, key)
}

func Int64Field(data map[string]any, key string) (int64, bool) {
	return nsmodel.Int64Field(data, key)
}

func BoolField(data map[string]any, key string) (bool, bool) {
	return nsmodel.BoolField(data, key)
}

func ToUTCString(value string) (string, int, error) {
	return nsmodel.ToUTCString(value)
}

func NormalizeCollection(collection string) string {
	return nsmodel.NormalizeCollection(collection)
}

func PathValue(data map[string]any, path string) any {
	return nsmodel.PathValue(data, path)
}
