package handlers

import (
	"SiteChecker/models"
	"context"
	"net/http"
	"time"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err := models.Mongo.Ping(ctx, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":    err == nil,
		"mongo": err == nil,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		"uptime": "",
	})
}
