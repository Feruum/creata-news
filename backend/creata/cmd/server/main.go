package main

import (
	"context"
	"creata.news/api/internal/news"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	seed := flag.String("seed", "", "Import development seed JSON (development only)")
	migrate := flag.Bool("migrate-only", false, "Apply Creata migrations and exit")
	grant := flag.String("grant-admin-email", "", "Grant administrator and chief editor to an existing verified-email account")
	importDir := flag.String("import-content", os.Getenv("CREATA_CONTENT_IMPORT_DIR"), "Import original approved content/assets once from a backend content directory")
	importOnly := flag.Bool("import-only", false, "Import original content and exit")
	testSchema := flag.String("test-schema", "", "Use a generated isolated verification schema (test environment/local test database only)")
	dropTestSchema := flag.String("drop-test-schema", "", "Drop exactly one generated verification schema and exit (test environment only)")
	flag.Parse()
	cfg := news.ConfigFromEnv()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *dropTestSchema != "" {
		if _, e := news.IsolatedTestSchema(ctx, cfg.Environment, cfg.DatabaseURL, *dropTestSchema, true); e != nil {
			log.Fatal(e)
		}
		return
	}
	if !*migrate && *grant == "" {
		if e := cfg.Validate(); e != nil {
			log.Fatal(e)
		}
	}
	if *testSchema != "" {
		var e error
		cfg.DatabaseURL, e = news.IsolatedTestSchema(ctx, cfg.Environment, cfg.DatabaseURL, *testSchema, false)
		if e != nil {
			log.Fatal(e)
		}
	}
	store, e := news.Open(ctx, cfg.DatabaseURL)
	if e != nil {
		log.Fatal(e)
	}
	defer store.Pool.Close()
	if e = store.Initialize(ctx); e != nil {
		log.Fatal(e)
	}
	if *seed != "" {
		if cfg.Environment == "production" {
			log.Fatal("development seed is forbidden in production")
		}
		b, e := os.ReadFile(*seed)
		if e != nil {
			log.Fatal(e)
		}
		if e = store.Seed(ctx, b); e != nil {
			log.Fatal(e)
		}
	}
	if *grant != "" {
		if e = store.GrantAdmin(ctx, *grant); e != nil {
			log.Fatal(e)
		}
		log.Print("Staff roles assigned to verified account")
		return
	}
	if *migrate {
		return
	}
	app, e := news.New(store, cfg)
	if e != nil {
		log.Fatal(e)
	}
	if *importDir != "" {
		b, e := os.ReadFile(filepath.Join(*importDir, "pages.json"))
		if e != nil {
			log.Fatal(e)
		}
		var bundle news.Doc
		if e = json.Unmarshal(b, &bundle); e != nil {
			log.Fatal(e)
		}
		if e = app.ImportContent(ctx, bundle, filepath.Join(*importDir, "assets")); e != nil {
			log.Fatal(e)
		}
		log.Print("Original Creata content imported; existing editorial revisions preserved")
	}
	if *importOnly {
		return
	}
	go app.RunScheduler(ctx)
	go app.RunPostiz(ctx)
	server := &http.Server{Addr: cfg.Address, Handler: app, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("Creata API listening on %s", cfg.Address)
	if e = server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
