package main

import (
	"SiteChecker/handlers"
	"log"
	"net/http"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/scan", handlers.ScanHandler)
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}
	log.Println("listening on :8080")
	log.Fatal(srv.ListenAndServe())

}
