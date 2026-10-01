package pgx

import (
	"bytes"
	"context"
	"database/sql/driver"
	"log/slog"
	"strings"
	"testing"

	"github.com/XSAM/otelsql"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type secretRow struct {
	ID       uint
	Password string
}

func openLogged(t *testing.T, cfg Config) (*gorm.DB, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: buildLogger(cfg)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&secretRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, &buf
}

func TestLogger_HidesBindValuesByDefault(t *testing.T) {
	db, buf := openLogged(t, Config{LogLevel: "info"})
	db.Create(&secretRow{Password: "hunter2"})

	if strings.Contains(buf.String(), "hunter2") {
		t.Fatalf("bind value leaked into logs: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "INSERT INTO") {
		t.Fatalf("query not logged at info: %q", buf.String())
	}
}

func TestLogger_LogSQLParamsOptIn(t *testing.T) {
	db, buf := openLogged(t, Config{LogLevel: "info", LogSQLParams: true})
	db.Create(&secretRow{Password: "visible-value"})

	if !strings.Contains(buf.String(), "visible-value") {
		t.Fatalf("expected bind values with LogSQLParams: %q", buf.String())
	}
}

func TestLogger_IgnoresRecordNotFound(t *testing.T) {
	db, buf := openLogged(t, Config{LogLevel: "warn"})
	var row secretRow
	if err := db.First(&row, 999).Error; err == nil {
		t.Fatal("expected ErrRecordNotFound")
	}
	if strings.Contains(buf.String(), "gorm error") {
		t.Fatalf("record-not-found logged as error: %q", buf.String())
	}

	db, buf = openLogged(t, Config{LogLevel: "warn", LogRecordNotFound: true})
	_ = db.First(&row, 999).Error
	if !strings.Contains(buf.String(), "gorm error") {
		t.Fatalf("LogRecordNotFound did not log: %q", buf.String())
	}
}

func TestOpenTracedWrapsDriver(t *testing.T) {
	db, err := openTraced("postgres://u:p@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("openTraced() error = %v", err)
	}
	defer db.Close()
	if _, ok := db.Driver().(interface {
		Open(string) (driver.Conn, error)
	}); !ok {
		t.Fatalf("unexpected driver %T", db.Driver())
	}
	if _, err := openTraced("://bad"); err == nil {
		t.Fatal("expected invalid dsn error")
	}
}

func TestTracedSpansOmitBindValues(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	sqlDB, err := otelsql.Open("sqlite3", "file::memory:", append(otelOptions(), otelsql.WithTracerProvider(tp))...)
	if err != nil {
		t.Fatalf("otelsql.Open: %v", err)
	}
	defer sqlDB.Close()
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{Logger: buildLogger(Config{LogLevel: "silent"})})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	if err := db.AutoMigrate(&secretRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db.WithContext(context.Background()).Create(&secretRow{Password: "span-secret"})

	var sawInsert bool
	for _, s := range recorder.Ended() {
		for _, kv := range s.Attributes() {
			v := kv.Value.Emit()
			if strings.Contains(v, "span-secret") {
				t.Fatalf("bind value leaked into span attribute %s", kv.Key)
			}
			if kv.Key == "db.query.text" && strings.Contains(v, "INSERT INTO") {
				sawInsert = true
			}
		}
	}
	if !sawInsert {
		t.Fatal("expected an INSERT span with db.query.text")
	}
}
