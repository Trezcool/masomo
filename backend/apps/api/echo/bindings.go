package echoapi

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/trezcool/masomo/core"
)

var orderingParam = "ordering"

type Ordering struct {
	Orderings []core.DBOrdering
}

func (ord *Ordering) Bind(ctx echo.Context) {
	data := ctx.QueryParams()
	if len(data) == 0 {
		return
	}
	val, ok := data[orderingParam]
	if !ok || len(val) == 0 || val[0] == "" {
		return
	}

	for _, field := range strings.Split(val[0], ",") {
		field = strings.TrimSpace(field)
		descending := strings.HasPrefix(field, "-")
		if descending {
			field = field[1:] // drop "-"
		}
		ord.Orderings = append(ord.Orderings, core.DBOrdering{Field: field, Ascending: !descending})
	}
}
