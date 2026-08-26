package llm_interaction

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type genPost struct {
	Handle   string `json:"handle"`
	Content  string `json:"content"`
	MediaURL string `json:"media_url"`
}

type postBot struct{ Handle, Name, Bio string }

func CreatePosts(db *sql.DB, n int) error {
	if n <= 0 {
		return nil
	}
	rows, err := db.Query(`SELECT handle, display_name, bio FROM bots`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var bots []postBot
	for rows.Next() {
		var b postBot
		rows.Scan(&b.Handle, &b.Name, &b.Bio)
		bots = append(bots, b)
	}
	if len(bots) == 0 {
		return fmt.Errorf("no bots for posts")
	}

	seenKW := map[string]bool{}
	seenImg := map[string]bool{}
	for done := 0; done < n; {
		batch := 10
		if n-done < batch {
			batch = n - done
		}
		var sb strings.Builder
		for _, b := range bots {
			fmt.Fprintf(&sb, "- @%s (%s): %s\n", b.Handle, b.Name, b.Bio)
		}
		prompt := fmt.Sprintf(`Generate %d social posts for fictional parody bots on a transparent satirical site. Each bot is openly labeled as fictional/parody (not a real person). Assign each post to one bot handle. Satirical parody content: hot takes, memes, gaming, manosphere, etc. Sometimes reference current news/trends if fits persona.

Rules: content max 280 chars (Twitter), may contain ONE https:// link. media_url "" or ONE https://loremflickr.com/600/400/<common-keyword> where keyword is single common word highly relevant to post image and MUST be common Flickr tag (e.g., gaming, cat, rain, gym, protest, food, beach, car, concert, meme) - map obscure (valorant->gaming, manosphere->gym). Only ~30%% of posts should have media, rest "" for variety. Vary keywords - used %v.

Bots:
%s
Return JSON array only: [{"handle":"...","content":"...","media_url":""}]`, batch, seenKW, sb.String())

		raw, err := callLLM(prompt)
		if err != nil {
			return err
		}
		js := extractJSON(raw)
		var posts []genPost
		if err := json.Unmarshal([]byte(js), &posts); err != nil {
			fmt.Printf("posts json fail %v raw %s\n", err, raw)
			continue
		}
		for _, p := range posts {
			if done >= n {
				break
			}
			p.Handle = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(p.Handle)), "@")
			if len([]rune(p.Content)) > 280 {
				p.Content = string([]rune(p.Content)[:280])
			}
			p.Content = strings.TrimSpace(p.Content)
			if p.Content == "" {
				continue
			}
			if p.MediaURL != "" {
				kw := extractKeyword(p.MediaURL)
				if kw == p.MediaURL || kw == "" {
					kw = keywordFromMediaURL(p.MediaURL)
				}
				if seenKW[kw] {
					kw = fmt.Sprintf("%s %s", kw, p.Handle)
				}
				seenKW[kw] = true
				img := searxImage(kw, seenImg)
				if img == "" {
					p.MediaURL = ""
				} else {
					p.MediaURL = img
				}
			}
			var botID int64
			if err := db.QueryRow(`SELECT id FROM bots WHERE handle=?`, p.Handle).Scan(&botID); err != nil {
				botID = bots[rand.Intn(len(bots))].id(db)
			}
			created := time.Now().Add(-time.Duration(rand.Intn(7*24)) * time.Hour).Add(-time.Duration(rand.Intn(60)) * time.Minute)
			_, err := db.Exec(`INSERT INTO posts(bot_id, content, media_url, created_at) VALUES(?,?,?,?)`,
				botID, p.Content, p.MediaURL, created.Format(time.RFC3339))
			if err != nil {
				fmt.Printf("post insert fail %v\n", err)
				continue
			}
			done++
			fmt.Printf("post %d/%d @%s %.40s media:%v\n", done, n, p.Handle, p.Content, p.MediaURL != "")
		}
	}
	return nil
}

func extractKeyword(url string) string {
	parts := strings.Split(url, "/")
	for i, p := range parts {
		if p == "400" && i+1 < len(parts) {
			k := strings.Split(parts[i+1], "?")[0]
			if k != "" {
				return k
			}
		}
		if p == "seed" && i+1 < len(parts) {
			return strings.Split(parts[i+1], "/")[0]
		}
	}
	if idx := strings.Index(url, "?"); idx != -1 {
		q := url[idx+1:]
		q = strings.Split(q, "&")[0]
		if q != "" {
			return q
		}
	}
	return url
}

func (b postBot) id(db *sql.DB) int64 {
	var id int64
	db.QueryRow(`SELECT id FROM bots WHERE handle=?`, b.Handle).Scan(&id)
	return id
}
