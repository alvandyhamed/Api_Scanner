package functions

import (
	"SiteChecker/models"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

	// 0) گروه‌بندی داخلی/خارجی
	inEP, extEP := splitInternalExternal(endpoints, host)
	inRES, extRES := splitInternalExternal(resources, host)
	// scripts خارجی/داخلی
	inSCR, extSCR := splitInternalExternal(scriptURLs, host)
	// ساخت externals map
	externals := makeExternalsMap(extEP, extRES, extSCR)

	// 1) Site upsert
	_, err = models.SitesColl().UpdateByID(ctx, siteID,
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
		return err
	}

	groupsEP := groupPaths(inEP, host)
	groupsRES := groupPaths(inRES, host)

	_, err = models.PagesColl().UpdateOne(ctx,
		bson.M{"url_norm": urlNorm},
		bson.M{
			"$set": bson.M{
				"site_id":         siteID,
				"scheme":          scheme,
				"host":            host,
				"path":            p,
				"url":             rawURL,
				"url_norm":        urlNorm,
				"resources":       inRES,
				"script_urls":     inSCR,
				"endpoints":       inEP,
				"groups":          groupsEP,
				"resource_groups": groupsRES,
				"externals":       externals,
				"scanned_at":      now,
			},
			"$setOnInsert": bson.M{"created_at": now},
		},
		mopts.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	for _, ep := range inEP {
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
		if _, err := models.EndpointsColl().UpdateOne(ctx, filter, update, mopts.Update().SetUpsert(true)); err != nil {
			return err
		}
	}

	sinks, err := ScanSinks(ctx, urlNorm, siteID)
	if err == nil && len(sinks) > 0 {

		for _, s := range sinks {
			_, _ = models.SinksColl().UpdateOne(ctx,
				bson.M{
					"site_id": s.SiteID, "page_url": s.PageURL,
					"source_url": s.SourceURL, "kind": s.Kind,
					"line": s.Line, "col": s.Col, "snippet": s.Snippet,
				},
				bson.M{
					"$setOnInsert": s,
					"$set":         bson.M{"detected_at": time.Now()},
				},
				mopts.Update().SetUpsert(true),
			)
		}
	}

	return nil
}

// --- helpers ---

func splitInternalExternal(items []string, pageHost string) (internal []string, extern map[string][]string) {
	extern = make(map[string][]string)
	for _, it := range uniqueStrings(items) {
		if it == "" {
			continue
		}

		iu, err := url.Parse(it)
		if err == nil && iu.IsAbs() {
			if !sameETLDPlusOne(iu.Hostname(), pageHost) {
				key := eTLD1(iu.Hostname())
				extern[key] = append(extern[key], it)
				continue
			}
			internal = append(internal, it)
			continue
		}

		internal = append(internal, it)
	}
	return internal, extern
}

func makeExternalsMap(extEP, extRES map[string][]string, extSCR map[string][]string) map[string]models.ExternalGroup {
	out := map[string]models.ExternalGroup{}
	for k, v := range extEP {
		eg := out[k]
		eg.SiteID = k
		eg.Endpoints = append(eg.Endpoints, uniqueStrings(v)...)
		out[k] = eg
	}
	for k, v := range extRES {
		eg := out[k]
		eg.SiteID = k
		eg.Resources = append(eg.Resources, uniqueStrings(v)...)
		out[k] = eg
	}
	for k, v := range extSCR {
		eg := out[k]
		eg.SiteID = k
		eg.Scripts = append(eg.Scripts, uniqueStrings(v)...)
		out[k] = eg
	}

	for k, eg := range out {
		hostset := map[string]struct{}{}
		for _, arr := range [][]string{eg.Endpoints, eg.Resources, eg.Scripts} {
			for _, u := range arr {
				if pu, err := url.Parse(u); err == nil && pu.Host != "" {
					hostset[pu.Hostname()] = struct{}{}
				}
			}
		}
		hosts := make([]string, 0, len(hostset))
		for h := range hostset {
			hosts = append(hosts, h)
		}
		eg.Hosts = hosts
		out[k] = eg
	}
	return out
}

func sameETLDPlusOne(a, b string) bool {
	return eTLD1(a) == eTLD1(b)
}
func eTLD1(h string) string {
	if h == "" {
		return ""
	}
	if base, err := publicsuffix.EffectiveTLDPlusOne(strings.ToLower(h)); err == nil && base != "" {
		return base
	}
	return strings.ToLower(h)
}

func groupPaths(items []string, host string) map[string][]string {
	out := map[string][]string{
		"api": {}, "javascript": {}, "html": {}, "asp": {}, "aspx": {}, "php": {}, "jsp": {}, "others": {}, "routes": {},
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

func categorize(host, pth string) string {
	s := strings.ToLower(pth)
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
func sinkSig(siteID, pageURL, sourceURL, kind string, line, col int, snippet string) string {
	if len(snippet) > 1000 {
		snippet = snippet[:1000]
	}
	sum := sha256.Sum256([]byte(siteID + "\x1f" + pageURL + "\x1f" + sourceURL + "\x1f" + kind +
		"\x1f" + fmt.Sprintf("%d:%d", line, col) + "\x1f" + snippet))
	return hex.EncodeToString(sum[:])
}

func PersistSinks(ctx context.Context, sinks []models.SinkDoc) (*mongo.BulkWriteResult, error) {
	if len(sinks) == 0 {
		return &mongo.BulkWriteResult{}, nil
	}

	// دِدوپ داخل همین batch
	uniq := make(map[string]models.SinkDoc, len(sinks))
	for _, s := range sinks {
		if s.SiteID == "" || s.PageURL == "" {
			continue
		}
		if len(s.Snippet) > 1000 {
			s.Snippet = s.Snippet[:1000]
		}
		sig := sinkSig(s.SiteID, s.PageURL, s.SourceURL, s.Kind, s.Line, s.Col, s.Snippet)
		sn := s
		// می‌تونی sig رو هم تو سند نگه داری
		uniq[sig] = sn
	}

	if len(uniq) == 0 {
		return &mongo.BulkWriteResult{}, nil
	}

	now := time.Now()
	modelsBW := make([]mongo.WriteModel, 0, len(uniq))

	for sig, s := range uniq {
		filter := bson.M{"sig": sig}
		update := bson.M{
			"$setOnInsert": bson.M{
				"sig":               sig,
				"site_id":           s.SiteID,
				"page_url":          s.PageURL,
				"source_url":        s.SourceURL,
				"kind":              s.Kind,
				"line":              s.Line,
				"col":               s.Col,
				"snippet":           s.Snippet,
				"first_detected_at": s.DetectedAt,
			},
			"$set": bson.M{
				"last_detected_at": now,
				"source_type":      s.SourceType, // فقط اینجا
			},
			"$inc": bson.M{
				"hits": 1, // فقط اینجا
			},
		}
		modelsBW = append(modelsBW, mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true))
	}

	opts := mopts.BulkWrite().SetOrdered(false)
	return models.SinksColl().BulkWrite(ctx, modelsBW, opts)
}
