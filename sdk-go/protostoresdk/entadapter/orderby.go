package entadapter

import (
	"entgo.io/ent/dialect/sql"

	"github.com/datakit-dev/dtkt-sdk/sdk-go/protostoresdk/orderby"
)

// EntOrder wraps an ent generated `By<Field>(...sql.OrderTermOption) O`
// helper into the FieldHandler shape expected by orderby.Apply,
// translating the parsed orderby.Direction into the matching ent order
// term option (`sql.OrderAsc()` / `sql.OrderDesc()`).
//
// Usage in a list handler:
//
//	fields := orderby.Fields[connection.OrderOption]{
//	    "name":       entadapter.EntOrder(connection.ByName),
//	    "created_at": entadapter.EntOrder(connection.ByCreatedAt),
//	    "updated_at": entadapter.EntOrder(connection.ByUpdatedAt),
//	    "id":         entadapter.EntOrder(connection.ByID),
//	}
//	terms, err := orderby.Apply(req.Msg.GetOrderBy(), fields)
//
// The result is a `[]connection.OrderOption` that can be assigned to
// PaginateOptions.OrderTerms.
func EntOrder[O ~func(*sql.Selector)](by func(...sql.OrderTermOption) O) orderby.FieldHandler[O] {
	return func(dir orderby.Direction) O {
		if dir == orderby.Desc {
			return by(sql.OrderDesc())
		}
		return by(sql.OrderAsc())
	}
}
