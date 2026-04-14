package ui

import (
	"context"
	"strconv"
	"time"

	"github.com/glycoview/glycoview/internal/model"
	"github.com/glycoview/glycoview/internal/store"
)

func (s Service) loadEntries(ctx context.Context, start, end time.Time) ([]model.Record, error) {
	return s.Store.Search(ctx, "entries", store.Query{
		Filters: []store.Filter{
			{Field: "date", Op: "gte", Value: strconv.FormatInt(start.UnixMilli(), 10)},
			{Field: "date", Op: "lte", Value: strconv.FormatInt(end.UnixMilli(), 10)},
		},
		Limit:     50000,
		SortField: "date",
		SortDesc:  false,
	})
}

func (s Service) loadTreatments(ctx context.Context, start, end time.Time) ([]model.Record, error) {
	return s.Store.Search(ctx, "treatments", store.Query{
		Filters: []store.Filter{
			{Field: "date", Op: "gte", Value: strconv.FormatInt(start.UnixMilli(), 10)},
			{Field: "date", Op: "lte", Value: strconv.FormatInt(end.UnixMilli(), 10)},
		},
		Limit:     10000,
		SortField: "date",
		SortDesc:  false,
	})
}

func (s Service) loadStatuses(ctx context.Context, start, end time.Time) ([]model.Record, error) {
	return s.Store.Search(ctx, "devicestatus", store.Query{
		Filters: []store.Filter{
			{Field: "date", Op: "gte", Value: strconv.FormatInt(start.UnixMilli(), 10)},
			{Field: "date", Op: "lte", Value: strconv.FormatInt(end.UnixMilli(), 10)},
		},
		Limit:     5000,
		SortField: "date",
		SortDesc:  false,
	})
}

func (s Service) latestProfile(ctx context.Context) (model.Record, error) {
	records, err := s.Store.Search(ctx, "profile", store.Query{
		Limit:     1,
		SortField: "srvModified",
		SortDesc:  true,
	})
	if err != nil || len(records) == 0 {
		return model.Record{}, err
	}
	return records[0], nil
}
