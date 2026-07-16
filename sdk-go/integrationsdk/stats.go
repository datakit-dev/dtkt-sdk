package integrationsdk

import (
	"encoding/json"
	"encoding/json/jsontext"
	"log/slog"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/util"
)

const StatsPath = "/_/stats"

func StatsHandler(log *slog.Logger, instances *util.SyncMap[string, *instance]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := jsontext.NewEncoder(w)
		err := enc.WriteToken(jsontext.BeginArray)
		if err != nil {
			log.Error(err.Error())
			return
		}

		instances.Range(func(hash string, inst *instance) bool {
			b, err := json.Marshal(map[string]any{
				"hash":        hash,
				"name":        inst.ResourceName(),
				"create_time": inst.createTime.String(),
			})
			if err != nil {
				log.Error(err.Error())
			} else {
				err = enc.WriteValue(b)
				if err != nil {
					log.Error(err.Error())
				}
			}

			return true
		})

		err = enc.WriteToken(jsontext.EndArray)
		if err != nil {
			log.Error(err.Error())
			return
		}
	})
}
