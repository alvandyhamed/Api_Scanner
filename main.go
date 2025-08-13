package main

import (
	"SiteChecker/functions"
	"SiteChecker/handlers"
	"SiteChecker/models"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := models.InitMongo(rootCtx); err != nil {
		log.Fatal("mongo init error: ", err)
	}
	if err := models.EnsureIndexes(rootCtx); err != nil {
		log.Println("mongo index warn: ", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = models.Mongo.Disconnect(ctx)
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/scan", handlers.ScanHandler)
	mux.HandleFunc("/api/scan", handlers.ScanHandler)

	mux.HandleFunc("/api/health", handlers.WithCORS(handlers.HealthHandler))

	mux.HandleFunc("/api/sites", handlers.WithCORS(handlers.SitesListHandler))

	mux.HandleFunc("/api/pages", handlers.WithCORS(handlers.PagesListHandler))
	mux.HandleFunc("/api/pages/by-url", handlers.WithCORS(handlers.PageByURLHandler))

	mux.HandleFunc("/api/endpoints", handlers.WithCORS(handlers.EndpointsListHandler))
	mux.HandleFunc("/api/endpoints/stats", handlers.WithCORS(handlers.EndpointsStatsHandler))

	mux.HandleFunc("/api/sinks", handlers.WithCORS(handlers.SinksListHandler))
	mux.HandleFunc("/api/sinks/stats", handlers.WithCORS(handlers.SinksStatsHandler))

	mux.HandleFunc("/api/externals", handlers.WithCORS(handlers.ExternalsListHandler))
	mux.HandleFunc("/api/search", handlers.WithCORS(handlers.SearchHandler))

	mux.HandleFunc("/api/watches", handlers.WithCORS(handlers.WatchesListHandler))           // GET
	mux.HandleFunc("/api/watches/create", handlers.WithCORS(handlers.WatchCreateHandler))    // POST
	mux.HandleFunc("/api/watches/scan-now", handlers.WithCORS(handlers.WatchScanNowHandler)) // POST
	mux.HandleFunc("/api/watches/delete", handlers.WithCORS(handlers.WatchDeleteHandler))

	mux.HandleFunc("/api/settings/discord", handlers.WithCORS(handlers.DiscordGetHandler))     // GET
	mux.HandleFunc("/api/settings/discord/set", handlers.WithCORS(handlers.DiscordSetHandler)) // POST
	mux.HandleFunc("/api/settings/discord/test", handlers.WithCORS(handlers.DiscordTestHandler))

	srv := &http.Server{
		Addr:         ":8050",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	go func() {
		log.Println("listening on :8050")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-rootCtx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	functions.StartWatchScheduler(ctx)
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
	log.Println("server stopped")

	//log.Fatal(srv.ListenAndServe())

}
