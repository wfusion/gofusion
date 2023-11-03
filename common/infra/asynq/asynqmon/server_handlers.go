package asynqmon

import (
	"net/http"

	"github.com/wfusion/gofusion/common/infra/asynq"
	"github.com/wfusion/gofusion/common/utils/serialize/json"
)

// ****************************************************************************
// This file defines:
//   - http.Handler(s) for server related endpoints
// ****************************************************************************

type listServersResponse struct {
	Servers []*serverInfo `json:"servers"`
}

func newListServersHandlerFunc(inspector *asynq.Inspector, pf PayloadFormatter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srvs, err := inspector.Servers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp := listServersResponse{
			Servers: toServerInfoList(srvs, pf),
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
