package functions

import (
	"SiteChecker/models"
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	mopts "go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/publicsuffix"
)

func SaveScanResults(ctx context.Context, rawURL string, resources, endpoints, scriptURLs []string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())

	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}

	urlNorm := scheme + "://" + u.Host + path
	if u.RawQuery != "" {
		urlNorm += "?" + u.RawQuery
	}

	base, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || base == "" {
		base = host
	}
	siteID := base
	now := time.Now()

	// 1) Site upsert
	resSite, err := models.SitesColl().UpdateByID(ctx, siteID,
		bson.M{
			"$set": bson.M{
				"updated_at":   now,
				"last_scan_at": now,
				"display_url":  scheme + "://" + base,
			},
			"$addToSet":    bson.M{"hosts": host},
			"$setOnInsert": bson.M{"created_at": now},
		},
		mopts.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("[mongo sites ERR] id=%s err=%v", siteID, err)
		return err
	}
	log.Printf("[mongo sites] matched=%d modified=%d upserted=%v",
		resSite.MatchedCount, resSite.ModifiedCount, resSite.UpsertedID)

	// 2) Page upsert
	resPage, err := models.PagesColl().UpdateOne(ctx,
		bson.M{"url_norm": urlNorm},
		bson.M{
			"$set": bson.M{
				"site_id":     siteID,
				"scheme":      scheme,
				"host":        host,
				"path":        path,
				"url":         rawURL,
				"url_norm":    urlNorm,
				"resources":   resources,
				"script_urls": scriptURLs,
				"endpoints":   endpoints,
				"scanned_at":  now,
			},
			"$setOnInsert": bson.M{"created_at": now},
		},
		mopts.Update().SetUpsert(true),
	)
	if err != nil {
		log.Printf("[mongo pages ERR] url_norm=%s err=%v", urlNorm, err)
		return err
	}
	log.Printf("[mongo pages] matched=%d modified=%d upserted=%v",
		resPage.MatchedCount, resPage.ModifiedCount, resPage.UpsertedID)

	// 3) Endpoints upsert (aggregate counts)
	var upserted, matched, modified int64
	for _, ep := range endpoints {
		resEp, err := models.EndpointsColl().UpdateOne(ctx,
			bson.M{"site_id": siteID, "endpoint": ep},
			bson.M{
				"$setOnInsert": bson.M{"first_seen": now},
				"$set":         bson.M{"last_seen": now},
				"$inc":         bson.M{"seen_count": 1},
				"$addToSet":    bson.M{"hosts": host},
				"$push":        bson.M{"source_urls": bson.M{"$each": []string{urlNorm}, "$slice": -5}},
			},
			mopts.Update().SetUpsert(true),
		)
		if err != nil {
			log.Printf("[mongo endpoints ERR] site=%s ep=%s err=%v", siteID, ep, err)
			return err
		}
		matched += resEp.MatchedCount
		modified += resEp.ModifiedCount
		if resEp.UpsertedID != nil {
			upserted++
		}
	}
	log.Printf("[mongo endpoints] total matched=%d modified=%d upserted=%d", matched, modified, upserted)

	return nil
}
