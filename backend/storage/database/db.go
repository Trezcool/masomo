package database

import (
	"database/sql"
	"net/url"

	_ "github.com/lib/pq"

	"github.com/trezcool/masomo/core"
)

func Open() (*sql.DB, error) {
	sslMode := "require"
	if core.Conf.Database.DisableTLS {
		sslMode = "disable"
	}
	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   core.Conf.Database.Engine,
		User:     url.UserPassword(core.Conf.Database.User, core.Conf.Database.Password),
		Host:     core.Conf.Database.Host,
		Path:     core.Conf.Database.Name,
		RawQuery: q.Encode(),
	}
	return sql.Open(core.Conf.Database.Engine, u.String())
}
