// Package postgres enables the PostgreSQL backend for the audit standard YAML
// factory without making the default standard import link the Postgres driver.
package postgres

import (
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/audit"
	auditstandard "github.com/brainlet/brainkit/modules/audit/standard"
	auditpostgres "github.com/brainlet/brainkit/modules/audit/stores/postgres"
)

func init() {
	auditstandard.RegisterStore("postgres", func(_ bkmodule.BuildContext, y auditstandard.YAML) (audit.Store, error) {
		s, err := auditpostgres.New(y.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("audit: open postgres: %w", err)
		}
		return s, nil
	})
}
