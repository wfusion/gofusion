package asynqmon

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// ****************************************************************************
// This file defines:
//   - http.Handler(s) for scheduler entry related endpoints
// ****************************************************************************

func newListSchedulerEntriesHandlerFunc(inspector *asynq.Inspector, pf PayloadFormatter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := inspector.SchedulerEntries()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payload := make(map[string]any)
		if len(entries) == 0 {
			// avoid nil for the entries field in json output.
			payload["entries"] = make([]*schedulerEntry, 0)
		} else {
			payload["entries"] = toSchedulerEntries(entries, pf)
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

type listSchedulerEnqueueEventsResponse struct {
	Events []*schedulerEnqueueEvent `json:"events"`
}

func newListSchedulerEnqueueEventsHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entryID := mux.Vars(r)["entry_id"]
		pageSize, pageNum := getPageOptions(r)
		events, err := inspector.ListSchedulerEnqueueEvents(
			entryID, asynq.PageSize(pageSize), asynq.Page(pageNum))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := listSchedulerEnqueueEventsResponse{
			Events: toSchedulerEnqueueEvents(events),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
