package functions

import (
	"SiteChecker/models"
	"context"
	"log"
	"net/url"
	"path"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

	p := u.EscapedPath()
	if p == "" {
		p = "/"
	}

	urlNorm := scheme + "://" + u.Host + p
	if u.RawQuery != "" {
		urlNorm += "?" + u.RawQuery
	}

	base, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil || base == "" {
		base = host
	}
	siteID := base
	now := time.Now()

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

	groupsEP := groupPaths(endpoints, host)
	groupsRES := groupPaths(resources, host)

	resPage, err := models.PagesColl().UpdateOne(ctx,
		bson.M{"url_norm": urlNorm},
		bson.M{
			"$set": bson.M{
				"site_id":         siteID,
				"scheme":          scheme,
				"host":            host,
				"path":            p,
				"url":             rawURL,
				"url_norm":        urlNorm,
				"resources":       resources,
				"script_urls":     scriptURLs,
				"endpoints":       endpoints,
				"groups":          groupsEP,
				"resource_groups": groupsRES,
				"scanned_at":      now,
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

	for _, ep := range endpoints {
		cat := categorize(host, ep)

		filter := bson.M{"site_id": siteID, "endpoint": ep}

		update := mongo.Pipeline{
			{{"$set", bson.M{

				"first_seen": bson.M{"$ifNull": bson.A{"$first_seen", now}},
				"last_seen":  now,
				"seen_count": bson.M{"$add": bson.A{bson.M{"$ifNull": bson.A{"$seen_count", 0}}, 1}},
				"hosts":      bson.M{"$setUnion": bson.A{bson.M{"$ifNull": bson.A{"$hosts", bson.A{}}}, bson.A{host}}},

				"source_urls": bson.M{
					"$slice": bson.A{
						bson.M{
							"$setUnion": bson.A{
								bson.A{urlNorm},
								bson.M{"$ifNull": bson.A{"$source_urls", bson.A{}}},
							},
						},
						5,
					},
				},
				"category": cat,
			}}},
		}

		resEp, err := models.EndpointsColl().UpdateOne(ctx, filter, update, mopts.Update().SetUpsert(true))
		if err != nil {
			log.Printf("[mongo endpoints ERR] site=%s ep=%s err=%v", siteID, ep, err)
			return err
		}
		_ = resEp
	}

	return nil
}

func groupPaths(items []string, host string) map[string][]string {
	out := map[string][]string{
		"api":        {},
		"javascript": {},
		"html":       {},
		"asp":        {},
		"aspx":       {},
		"php":        {},
		"jsp":        {},
		"others":     {},
		"routes":     {},
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if it == "" {
			continue
		}
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		c := categorize(host, it)
		out[c] = append(out[c], it)
	}
	return out
}

func categorize(host, p string) string {
	s := strings.ToLower(p)

	base := s
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base = base[:i]
	}
	if j := strings.IndexByte(base, '#'); j >= 0 {
		base = base[:j]
	}
	ext := strings.ToLower(path.Ext(base))

	switch ext {
	case ".js":
		return "javascript"
	case ".html", ".htm":
		return "html"
	case ".asp":
		return "asp"
	case ".aspx":
		return "aspx"
	case ".php":
		return "php"
	case ".jsp":
		return "jsp"
	}

	if ext != "" {
		return "others"
	}

	if strings.HasPrefix(host, "api.") || strings.Contains(s, "/api") {
		return "api"
	}

	return "routes"
}
