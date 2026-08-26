package llm_interaction

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var searxBase = func() string {
	if v := os.Getenv("SEARXNG_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://searxng.home.arpa"
}()

var searxClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func searxImage(keyword string, seen map[string]bool) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}
	keyword = strings.Split(keyword, "?")[0]
	keyword = strings.Split(keyword, "&")[0]
	u := fmt.Sprintf("%s/search?q=%s&format=json&categories=images", searxBase, url.QueryEscape(keyword))
	resp, err := searxClient.Get(u)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var out struct {
		Results []struct {
			ImgSrc string `json:"img_src"`
			Engine string `json:"engine"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Results) == 0 {
		return ""
	}
	var candidates []string
	for _, r := range out.Results {
		if r.ImgSrc == "" || seen[r.ImgSrc] {
			continue
		}
		if strings.HasSuffix(strings.ToLower(r.ImgSrc), ".svg") || r.Engine == "lucide" {
			continue
		}
		// skip tiny icons, prefer jpg/png/webp
		low := strings.ToLower(r.ImgSrc)
		if !(strings.Contains(low, ".jpg") || strings.Contains(low, ".jpeg") || strings.Contains(low, ".png") || strings.Contains(low, ".webp")) {
			// allow if no extension but not svg
			if strings.Contains(low, ".svg") {
				continue
			}
		}
		candidates = append(candidates, r.ImgSrc)
		if len(candidates) >= 8 {
			break
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	pick := candidates[rand.Intn(len(candidates))]
	seen[pick] = true
	return pick
}

func keywordFromMediaURL(media string) string {
	if media == "" {
		return ""
	}
	k := extractKeyword(media)
	if k == media || k == "" {
		if idx := strings.Index(media, "?"); idx != -1 {
			k = strings.Split(media[idx+1:], "&")[0]
		}
	}
	k = strings.TrimSpace(k)
	parts := strings.Fields(k)
	if len(parts) > 2 {
		k = strings.Join(parts[:2], " ")
	}
	return k
}
