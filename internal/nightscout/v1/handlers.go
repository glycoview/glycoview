package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/better-monitoring/bscout/internal/auth"
	"github.com/better-monitoring/bscout/internal/config"
	"github.com/better-monitoring/bscout/internal/httpx"
	"github.com/better-monitoring/bscout/internal/model"
	"github.com/better-monitoring/bscout/internal/query"
	"github.com/better-monitoring/bscout/internal/store"
)

type Dependencies struct {
	Config config.Config
	Store  store.Store
	Auth   *auth.Manager
}

func NewRouter(dep Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Get("/status.json", statusJSON(dep))
	r.Get("/status.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("STATUS OK"))
	})
	r.Get("/status.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body>Status OK</body></html>"))
	})
	r.Get("/status.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = fmt.Fprintf(w, "this.serverSettings = %s;", statusJSONPayload(dep))
	})
	r.Get("/status.svg", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://img.shields.io/badge/Nightscout-OK-green.svg", http.StatusFound)
	})
	r.Get("/status.png", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://img.shields.io/badge/Nightscout-OK-green.png", http.StatusFound)
	})

	r.Get("/verifyauth", verifyAuth(dep))

	r.Get("/versions", versions())
	r.Get("/echo/{collection}/{spec}.json", dep.Auth.Require("api:entries:read", true, echoRoute(dep)))
	r.Get("/slice/{collection}/{field}/{type}/{prefix}.json", dep.Auth.Require("api:entries:read", true, sliceRoute(dep)))
	r.Get("/times/echo/*", dep.Auth.Require("api:entries:read", true, timesEchoRoute(dep)))
	r.Get("/times/*", dep.Auth.Require("api:entries:read", true, timesRoute(dep)))

	r.Get("/entries/current.json", dep.Auth.Require("api:entries:read", true, entriesCurrent(dep)))
	r.Get("/entries.json", dep.Auth.Require("api:entries:read", true, entriesList(dep)))
	r.Get("/entries/{spec}.json", dep.Auth.Require("api:entries:read", true, entriesSpec(dep)))
	r.Post("/entries", requireV1Write(dep, "api:entries:create", entriesCreate(dep, true)))
	r.Post("/entries/", requireV1Write(dep, "api:entries:create", entriesCreate(dep, true)))
	r.Post("/entries.json", requireV1Write(dep, "api:entries:create", entriesCreate(dep, true)))
	r.Post("/entries/preview.json", requireV1Write(dep, "api:entries:create", entriesCreate(dep, false)))
	r.Delete("/entries.json", requireV1Write(dep, "api:entries:delete", entriesDelete(dep)))
	r.Delete("/entries", requireV1Write(dep, "api:entries:delete", entriesDelete(dep)))
	r.Delete("/entries/{spec}", requireV1Write(dep, "api:entries:delete", entriesDelete(dep)))
	r.Get("/treatments", dep.Auth.Require("api:treatments:read", true, treatmentsList(dep)))
	r.Get("/treatments.json", dep.Auth.Require("api:treatments:read", true, treatmentsList(dep)))
	r.Post("/treatments", dep.Auth.Require("api:treatments:create", false, treatmentsCreate(dep)))
	r.Post("/treatments/", dep.Auth.Require("api:treatments:create", false, treatmentsCreate(dep)))
	r.Delete("/treatments", dep.Auth.Require("api:treatments:delete", false, treatmentsDelete(dep)))
	r.Delete("/treatments/", dep.Auth.Require("api:treatments:delete", false, treatmentsDelete(dep)))
	r.Get("/devicestatus", dep.Auth.Require("api:devicestatus:read", true, genericCollectionList(dep, "devicestatus", "created_at")))
	r.Get("/devicestatus.json", dep.Auth.Require("api:devicestatus:read", true, genericCollectionList(dep, "devicestatus", "created_at")))
	r.Post("/devicestatus", dep.Auth.Require("api:devicestatus:create", false, genericCollectionCreate(dep, "devicestatus")))
	r.Post("/devicestatus/", dep.Auth.Require("api:devicestatus:create", false, genericCollectionCreate(dep, "devicestatus")))
	r.Delete("/devicestatus", dep.Auth.Require("api:devicestatus:delete", false, genericCollectionDelete(dep, "devicestatus", "created_at")))
	r.Delete("/devicestatus/", dep.Auth.Require("api:devicestatus:delete", false, genericCollectionDelete(dep, "devicestatus", "created_at")))
	r.Get("/profile", dep.Auth.Require("api:profile:read", true, profileList(dep)))
	r.Get("/profile.json", dep.Auth.Require("api:profile:read", true, profileList(dep)))
	r.Get("/settings", dep.Auth.Require("api:settings:read", true, genericCollectionList(dep, "settings", "created_at")))
	r.Get("/settings.json", dep.Auth.Require("api:settings:read", true, genericCollectionList(dep, "settings", "created_at")))
	r.Get("/food", dep.Auth.Require("api:food:read", true, genericCollectionList(dep, "food", "created_at")))
	r.Get("/food.json", dep.Auth.Require("api:food:read", true, genericCollectionList(dep, "food", "created_at")))

	return r
}

func versions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, []map[string]string{
			{"url": "/api/v1", "version": "1.0.0"},
			{"url": "/api/v2", "version": "2.0.0"},
			{"url": "/api/v3", "version": "3.0.4"},
		})
	}
}

func verifyAuth(dep Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := dep.Auth.AuthenticateExplicit(r)
		if err != nil {
			httpx.WriteJSON(w, http.StatusOK, map[string]any{
				"status": "ok",
				"message": map[string]any{
					"message": "UNAUTHORIZED",
					"isAdmin": false,
				},
			})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"message": map[string]any{
				"message": "OK",
				"isAdmin": dep.Auth.HasPermission(*identity, "api:*:admin") || dep.Auth.HasPermission(*identity, "api:entries:create"),
			},
		})
	}
}

func requireV1Write(dep Dependencies, permission string, next func(http.ResponseWriter, *http.Request, *auth.Identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := dep.Auth.AuthenticateExplicit(r)
		if err != nil || identity == nil || !dep.Auth.HasPermission(*identity, permission) {
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]any{
				"status":      http.StatusUnauthorized,
				"message":     "Unauthorized",
				"description": "The requested operation requires API write authorization.",
			})
			return
		}
		next(w, r.WithContext(r.Context()), identity)
	}
}

func statusJSON(dep Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"apiEnabled":        true,
			"careportalEnabled": contains(dep.Config.Enable, "careportal"),
			"settings": map[string]any{
				"enable": dep.Config.Enable,
			},
		})
	}
}

func statusJSONPayload(dep Dependencies) string {
	return fmt.Sprintf(`{"apiEnabled":true,"careportalEnabled":%t,"settings":{"enable":["%s"]}}`, contains(dep.Config.Enable, "careportal"), strings.Join(dep.Config.Enable, `","`))
}

func entriesCurrent(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := store.DefaultQuery()
		query.Limit = 1
		records, err := dep.Store.Search(r.Context(), "entries", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeRecords(w, records, nil)
	}
}

func entriesList(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "date")
		records, err := dep.Store.Search(r.Context(), "entries", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeRecords(w, records, nil)
	}
}

func entriesSpec(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		spec := chi.URLParam(r, "spec")
		switch spec {
		case "current":
			entriesCurrent(dep)(w, r, identity)
			return
		case "sgv", "mbg":
			query := query.ParseV1(r.URL.Query(), "date")
			query.Filters = append(query.Filters, store.Filter{Field: "type", Op: "eq", Value: spec})
			records, err := dep.Store.Search(r.Context(), "entries", query)
			if err != nil {
				httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
				return
			}
			writeRecords(w, records, nil)
			return
		default:
			record, err := dep.Store.Get(r.Context(), "entries", spec)
			if err != nil {
				status, body := httpx.RequireRecord(err)
				httpx.WriteJSON(w, status, body)
				return
			}
			writeRecords(w, []model.Record{record}, nil)
		}
	}
}

func treatmentsList(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "created_at")
		records, err := dep.Store.Search(r.Context(), "treatments", query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeRecords(w, records, nil)
	}
}

func treatmentsCreate(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		switch typed := body.(type) {
		case map[string]any:
			if _, _, err := dep.Store.Create(r.Context(), "treatments", typed, identity.Name); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
		case []any:
			for _, item := range typed {
				doc, ok := item.(map[string]any)
				if !ok {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
					return
				}
				if _, _, err := dep.Store.Create(r.Context(), "treatments", doc, identity.Name); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		records, _ := dep.Store.Search(r.Context(), "treatments", store.DefaultQuery())
		writeRecords(w, records, nil)
	}
}

func treatmentsDelete(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "created_at")
		deleted, err := dep.Store.DeleteMatching(r.Context(), "treatments", query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}

func profileList(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		records, err := dep.Store.Search(r.Context(), "profile", store.Query{Limit: 100, SortField: "created_at", SortDesc: true})
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeRecords(w, records, nil)
	}
}

func entriesCreate(dep Dependencies, persist bool) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		preview := make([]map[string]any, 0)
		apply := func(doc map[string]any) error {
			doc = model.CloneMap(doc)
			if dateString, ok := model.StringField(doc, "dateString"); ok && dateString != "" {
				normalized, offset, err := model.ToUTCString(dateString)
				if err == nil {
					doc["dateString"] = normalized
					doc["utcOffset"] = offset
				}
			}
			if persist {
				record, _, err := dep.Store.Create(r.Context(), "entries", doc, identity.Name)
				if err == nil {
					preview = append(preview, record.ToMap(false))
				}
				return err
			}
			if _, ok := model.StringField(doc, "_id"); !ok {
				doc["_id"] = store.GenerateIdentifier()
			}
			if _, ok := model.StringField(doc, "identifier"); !ok {
				doc["identifier"] = doc["_id"]
			}
			preview = append(preview, doc)
			return nil
		}
		switch typed := body.(type) {
		case map[string]any:
			if err := apply(typed); err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
		case []any:
			for _, item := range typed {
				doc, ok := item.(map[string]any)
				if !ok {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
					return
				}
				if err := apply(doc); err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		status := http.StatusOK
		if !persist {
			status = http.StatusCreated
		}
		httpx.WriteJSON(w, status, preview)
	}
}

func entriesDelete(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), "date")
		if spec := chi.URLParam(r, "spec"); spec != "" && spec != "json" {
			query.Filters = append(query.Filters, store.Filter{Field: "type", Op: "eq", Value: spec})
		}
		deleted, err := dep.Store.DeleteMatching(r.Context(), "entries", query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}

func genericCollectionList(dep Dependencies, collection, defaultDateField string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), defaultDateField)
		records, err := dep.Store.Search(r.Context(), collection, query)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeRecords(w, records, nil)
	}
}

func genericCollectionCreate(dep Dependencies, collection string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		var body any
		if err := httpx.ReadJSON(r, &body); err != nil {
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		var created []model.Record
		switch typed := body.(type) {
		case map[string]any:
			record, _, err := dep.Store.Create(r.Context(), collection, typed, identity.Name)
			if err != nil {
				httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
				return
			}
			created = append(created, record)
		case []any:
			for _, item := range typed {
				doc, ok := item.(map[string]any)
				if !ok {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
					return
				}
				record, _, err := dep.Store.Create(r.Context(), collection, doc, identity.Name)
				if err != nil {
					httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest, "message": err.Error()})
					return
				}
				created = append(created, record)
			}
		default:
			httpx.WriteJSON(w, http.StatusBadRequest, map[string]any{"status": http.StatusBadRequest})
			return
		}
		writeRecords(w, created, nil)
	}
}

func genericCollectionDelete(dep Dependencies, collection, defaultDateField string) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		query := query.ParseV1(r.URL.Query(), defaultDateField)
		deleted, err := dep.Store.DeleteMatching(r.Context(), collection, query, true, identity.Name)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"status": http.StatusInternalServerError, "message": err.Error()})
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
	}
}

func echoRoute(_ Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		httpx.WriteJSON(w, http.StatusOK, query.EchoV1(chi.URLParam(r, "collection"), r.URL.Query()))
	}
}

func sliceRoute(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		collection := chi.URLParam(r, "collection")
		q := query.ParseV1(r.URL.Query(), "date")
		q.Filters = stripImplicitDateFilters(q.Filters)
		q.Limit = max(q.Limit, 1000)
		records, err := dep.Store.Search(r.Context(), collection, q)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		filtered := query.Slice(records, chi.URLParam(r, "field"), chi.URLParam(r, "type"), chi.URLParam(r, "prefix"), q.Limit)
		writeRecords(w, filtered, nil)
	}
}

func timesEchoRoute(_ Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		prefix, expr, ok := parseTimesWildcard(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, patterns := query.Times(nil, prefix, expr, 0)
		httpx.WriteJSON(w, http.StatusOK, map[string]any{
			"req":     map[string]any{"query": r.URL.Query()},
			"pattern": patterns,
		})
	}
}

func timesRoute(dep Dependencies) func(http.ResponseWriter, *http.Request, *auth.Identity) {
	return func(w http.ResponseWriter, r *http.Request, identity *auth.Identity) {
		prefix, expr, ok := parseTimesWildcard(r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		q := query.ParseV1(r.URL.Query(), "date")
		q.Filters = stripImplicitDateFilters(q.Filters)
		q.Limit = max(q.Limit, 1000)
		records, err := dep.Store.Search(r.Context(), "entries", q)
		if err != nil {
			httpx.WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		filtered, _ := query.Times(records, prefix, expr, q.Limit)
		writeRecords(w, filtered, nil)
	}
}

func writeRecords(w http.ResponseWriter, records []model.Record, fields []string) {
	response := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if len(fields) == 0 {
			response = append(response, record.ToMap(false))
		} else {
			response = append(response, store.SelectFields(record, fields))
		}
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stripImplicitDateFilters(filters []store.Filter) []store.Filter {
	out := make([]store.Filter, 0, len(filters))
	for _, filter := range filters {
		if filter.Field == "date" && filter.Op == "gte" && strings.Contains(filter.Value, "T") {
			continue
		}
		if filter.Field == "created_at" && filter.Op == "gte" && strings.Contains(filter.Value, "T") {
			continue
		}
		out = append(out, filter)
	}
	return out
}

func parseTimesWildcard(r *http.Request) (prefix, expr string, ok bool) {
	wild := chi.URLParam(r, "*")
	parts := strings.SplitN(wild, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	prefix = parts[0]
	expr = strings.TrimSuffix(parts[1], ".json")
	if decoded, err := url.PathUnescape(prefix); err == nil {
		prefix = decoded
	}
	if decoded, err := url.PathUnescape(expr); err == nil {
		expr = decoded
	}
	if prefix == "" || expr == "" {
		return "", "", false
	}
	return prefix, expr, true
}
