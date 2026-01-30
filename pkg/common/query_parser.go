package common

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var opMap = map[string]Operator{
	"eq":  OpEq,
	"ne":  OpNe,
	"gt":  OpGt,
	"gte": OpGte,
	"lt":  OpLt,
	"lte": OpLte,
	"in":  OpIn,
	"nin": OpNe,
	"re":  OpIn,
}

func ParseQueryArgs(args map[string]string) (*QuerySpec, error) {
	spec := &QuerySpec{}

	VisitAll(args, func(key, val string) {
		k := key
		v := val

		switch k {
		case "limit":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				spec.Limit = n
			}
		case "skip":
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				spec.Offset = n
			}
		case "sort":
			spec.Sort = append(spec.Sort, Sort{Field: v, Desc: false})
		case "sort$desc":
			spec.Sort = append(spec.Sort, Sort{Field: v, Desc: true})
		case "fields":

		default:
			// look for filter operator in key: <field>$<op>
			if strings.Contains(k, "$") {
				parts := strings.SplitN(k, "$", 2)
				field := parts[0]
				opStr := parts[1]

				// decode value (in case of encoded pipes)
				rawVal, _ := url.QueryUnescape(v)

				// special handling for IN/NIN (pipe separated)
				if opStr == "in" || opStr == "nin" {
					items := strings.Split(rawVal, "|")
					spec.Filters = append(spec.Filters, Filter{Field: field, Op: OpIn, Value: items})
					if opStr == "nin" {
						// represent NOT IN as negation in queries; keep OpNe for later handling
					}
					return
				}

				// date handling: normalize to milliseconds for certain fields
				if field == "date" || field == "srvModified" || field == "srvCreated" {
					if ts, err := parseDateToMillis(rawVal); err == nil {
						if mappedOp, ok := opMap[opStr]; ok {
							spec.Filters = append(spec.Filters, Filter{Field: field, Op: mappedOp, Value: ts})
						}
						return
					}
				}

				// regex maps to IN/REGEXP handling; store raw string
				if opStr == "re" {
					spec.Filters = append(spec.Filters, Filter{Field: field, Op: OpIn, Value: rawVal})
					return
				}

				// default: map operator and store value
				if mappedOp, ok := opMap[opStr]; ok {
					spec.Filters = append(spec.Filters, Filter{Field: field, Op: mappedOp, Value: rawVal})
				}
			}
		}
	})

	return spec, nil
}

func parseDateToMillis(v string) (int64, error) {
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		if n < 1e11 {
			return n * 1000, nil
		}
		return n, nil
	}

	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t.UTC().UnixNano() / int64(time.Millisecond), nil
		}
	}

	re := regexp.MustCompile(`\d+`)
	m := re.FindString(v)
	if m != "" {
		if n, err := strconv.ParseInt(m, 10, 64); err == nil {
			if n < 1e11 {
				return n * 1000, nil
			}
			return n, nil
		}
	}

	return 0, fmt.Errorf("unsupported date format: %s", v)
}
