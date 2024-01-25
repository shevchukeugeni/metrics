package migrations

import (
	"embed"
	"net/http"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
)

//go:embed *.sql
var static embed.FS

func init() {
	source.Register("embed", &driver{})
}

type driver struct {
	httpfs.PartialDriver
}

func (d *driver) Open(url string) (source.Driver, error) {
	err := d.PartialDriver.Init(http.FS(static), ".")
	if err != nil {
		return nil, err
	}
	return d, err
}
