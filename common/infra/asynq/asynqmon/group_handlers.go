package asynqmon

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

type listGroupsResponse struct {
	Queue  *queueStateSnapshot `json:"stats"`
	Groups []*groupInfo        `json:"groups"`
}

func newListGroupsHandlerFunc(inspector *asynq.Inspector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qname := mux.Vars(r)["qname"]

		groups, err := inspector.Groups(qname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		qinfo, err := inspector.GetQueueInfo(qname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := listGroupsResponse{
			Queue:  toQueueStateSnapshot(qinfo),
			Groups: toGroupInfos(groups),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
