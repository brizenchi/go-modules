package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	billingrepo "github.com/brizenchi/go-modules/modules/billing/adapter/repo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This one-off tool migrates Stripe columns from the pre-ADR-0001 users table
// into billing-owned tables. New SaaS projects do not need it.
func main() {
	var (
		mode   = flag.String("mode", "check", "mode: check or backfill")
		dsn    = flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres DSN (or set DATABASE_URL)")
		userID = flag.String("user-id", "", "Only process one user ID")
		limit  = flag.Int("limit", 0, "Maximum legacy rows to scan")
		asJSON = flag.Bool("json", false, "Print JSON output")
	)
	flag.Parse()

	if *dsn == "" {
		log.Fatal("dsn required: pass --dsn or set DATABASE_URL")
	}
	db, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := billingrepo.AutoMigrate(db); err != nil {
		log.Fatalf("migrate billing tables: %v", err)
	}

	options := billingrepo.LegacyBillingSyncOptions{UserID: *userID, Limit: *limit}
	ctx := context.Background()
	switch *mode {
	case "backfill":
		report, err := billingrepo.BackfillLegacyStripeState(ctx, db, options)
		if err != nil {
			log.Fatalf("backfill failed: %v", err)
		}
		printReport(*asJSON, report)
	case "check":
		report, err := billingrepo.CheckLegacyStripeState(ctx, db, options)
		if err != nil {
			log.Fatalf("check failed: %v", err)
		}
		printReport(*asJSON, report)
		if !report.OK() {
			os.Exit(2)
		}
	default:
		log.Fatalf("unsupported mode %q, want check or backfill", *mode)
	}
}

func printReport(asJSON bool, value any) {
	if asJSON {
		output, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			log.Fatalf("marshal json: %v", err)
		}
		fmt.Println(string(output))
		return
	}
	fmt.Printf("%+v\n", value)
}
