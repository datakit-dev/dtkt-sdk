package integration

import (
	"encoding/json"
	"encoding/json/jsontext"
	"net/http"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/integration/v1beta1"
)

const statsPath = "/_/stats"

func statsHandler[C any, I v1beta1.InstanceType](intgr *Integration[C, I]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enc := jsontext.NewEncoder(w)
		err := enc.WriteToken(jsontext.BeginArray)
		if err != nil {
			intgr.log.Error(err.Error())
			return
		}

		intgr.instances.Range(func(hash string, inst *instance[C, I]) bool {
			b, err := json.Marshal(map[string]any{
				"hash":        hash,
				"name":        inst.name,
				"create_time": inst.created.String(),
			})
			if err != nil {
				intgr.log.Error(err.Error())
			} else {
				err = enc.WriteValue(b)
				if err != nil {
					intgr.log.Error(err.Error())
				}
			}

			return true
		})

		err = enc.WriteToken(jsontext.EndArray)
		if err != nil {
			intgr.log.Error(err.Error())
			return
		}
	})
}
