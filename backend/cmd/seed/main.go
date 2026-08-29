// seed — creates or updates an admin member in WEO_MEMBER
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"github.com/dflh-saf/backend/internal/repository"
	"github.com/dflh-saf/backend/internal/service"
)

func main() {
	id := flag.String("id", "", "Login ID (USR_ID) — required")
	password := flag.String("password", "", "Plain-text password — required")
	name := flag.String("name", "관리자", "Display name (USR_NAME)")
	flag.Parse()

	if *id == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "Usage: seed --id <id> --password <password> [--name <name>]")
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL env required")
	}

	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	hashed := service.MysqlNativePassword(*password)

	_, err = repository.NewAdminMemberRepository(db).UpsertRootAdminMember(*id, hashed, *name)
	if err != nil {
		log.Fatal("seed failed:", err)
	}

	var result struct {
		Seq    int    `db:"USR_SEQ"`
		ID     string `db:"USR_ID"`
		Status string `db:"USR_STATUS"`
	}
	err = db.Get(&result, "SELECT USR_SEQ, USR_ID, USR_STATUS FROM WEO_MEMBER WHERE USR_ID = ?", *id)
	if err != nil {
		log.Fatal("select failed:", err)
	}

	fmt.Printf("OK — USR_SEQ=%d USR_ID=%s USR_STATUS=%s\n", result.Seq, result.ID, result.Status)
}
