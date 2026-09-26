package repository

import (
	"os"
	"testing"

	"github.com/dflh-saf/backend/internal/testsupport/mariadb"
)

func TestMain(m *testing.M) {
	os.Exit(mariadb.Run(m))
}
