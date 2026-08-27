package llm_interaction

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

type genComment struct {
	Handle   string `json:"handle"`
	PostID   int64  `json:"post_id"`
	ParentID *int64 `json:"parent_id"`
	Content  string `json:"content"`
	MediaURL string `json:"media_url"`
}

func CreateComments(db *sql.DB, n int) error {
	if n <= 0 {
		return nil
	}
	postRows, err := db.Query(`SELECT p.id, p.content, p.created_at, b.handle, b.display_name, b.bio, b.id FROM posts p JOIN bots b ON b.id=p.bot_id ORDER BY p.created_at DESC`)
	if err != nil {
		return err
	}
	defer postRows.Close()
	type postInfo struct{ ID int64; Content, Created string; Handle, Name, Bio string; BotID int64; Time time.Time }
	var posts []postInfo
	for postRows.Next() {
		var pi postInfo
		postRows.Scan(&pi.ID, &pi.Content, &pi.Created, &pi.Handle, &pi.Name, &pi.Bio, &pi.BotID)
		pi.Time, _ = time.Parse(time.RFC3339, pi.Created)
		if pi.Time.IsZero() {
			pi.Time, _ = time.Parse("2006-01-02 15:04:05", pi.Created)
		}
		posts = append(posts, pi)
	}
	if len(posts) == 0 {
		return fmt.Errorf("no posts for comments")
	}
	botRows, _ := db.Query(`SELECT handle, display_name, bio, id FROM bots`)
	defer botRows.Close()
	type botInfo struct{ Handle, Name, Bio string; ID int64 }
	var bots []botInfo
	for botRows.Next() {
		var b botInfo
		botRows.Scan(&b.Handle, &b.Name, &b.Bio, &b.ID)
		bots = append(bots, b)
	}
	botByHandle := map[string]int64{}
	for _, b := range bots {
		botByHandle[strings.ToLower(b.Handle)] = b.ID
	}

	seenKW := map[string]bool{}
	seenImg := map[string]bool{}
	for done := 0; done < n; {
		batch := 10
		if n-done < batch {
			batch = n - done
		}
		sampled := weightedPick(posts, batch)

		// build prompt with post + its existing thread context so LLM can reply to comments too
		var sb strings.Builder
		for _, p := range sampled {
			fmt.Fprintf(&sb, "Post %d by @%s: %s\n", p.ID, p.Handle, p.Content)
			cRows, _ := db.Query(`SELECT c.id, c.content, b.handle, COALESCE(c.parent_comment_id,0) FROM comments c JOIN bots b ON b.id=c.bot_id WHERE c.post_id=? ORDER BY c.created_at LIMIT 6`, p.ID)
			if cRows != nil {
				has := false
				for cRows.Next() {
					var cid, parent int64
					var ctext, chandle string
					cRows.Scan(&cid, &ctext, &chandle, &parent)
					if !has {
						fmt.Fprintf(&sb, "  Thread on post %d:\n", p.ID)
						has = true
					}
					if parent != 0 {
						fmt.Fprintf(&sb, "    - Comment %d by @%s (reply to %d): %s\n", cid, chandle, parent, ctext)
					} else {
						fmt.Fprintf(&sb, "    - Comment %d by @%s: %s\n", cid, chandle, ctext)
					}
				}
				cRows.Close()
				if !has {
					fmt.Fprintf(&sb, "  (no comments yet on this post)\n")
				}
			}
		}
		var botSb strings.Builder
		for _, b := range bots {
			fmt.Fprintf(&botSb, "- @%s (%s): %s\n", b.Handle, b.Name, b.Bio)
		}
		prompt := fmt.Sprintf(`Generate %d comments for fictional parody bots on transparent satirical site. Each bot openly labeled fictional, not real person. Comments max 280 chars, may contain ONE https:// link. media_url "" or ONE https://loremflickr.com/600/400/<common-keyword> where keyword is single common word highly relevant to comment image (e.g., gaming, cat, rain, gym). Must be common Flickr tag - map obscure to common. Only ~20%% of comments should have media, rest "".

Rules: each comment MUST directly respond to its post_id's post content — reference specifics, not generic. If thread exists for that post, ~40%% of comments should be replies to existing comments: set parent_id to that comment's id and make content reply to that comment's text (agree/disagree/joke/argue). parent_id null = top-level reply to post. ~60%% top-level, ~40%% threaded where threads exist.

Posts + threads:
%s
Bots:
%s
Return JSON array only: [{"handle":"...","post_id":123,"parent_id":null,"content":"...","media_url":""}] post_id must be one of the listed post ids, parent_id must be null or an existing comment id from SAME post's thread.`, batch, sb.String(), botSb.String())

		raw, err := callLLM(prompt)
		if err != nil {
			return err
		}
		js := extractJSON(raw)
		var comments []genComment
		if err := json.Unmarshal([]byte(js), &comments); err != nil {
			fmt.Printf("comments json fail %v raw %s\n", err, raw)
			continue
		}
		for _, c := range comments {
			if done >= n {
				break
			}
			c.Handle = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(c.Handle)), "@")
			if len([]rune(c.Content)) > 280 {
				c.Content = string([]rune(c.Content)[:280])
			}
			c.Content = strings.TrimSpace(c.Content)
			if c.Content == "" {
				continue
			}
			if c.MediaURL != "" {
				kw := extractKeyword(c.MediaURL)
				if kw == c.MediaURL {
					kw = keywordFromMediaURL(c.MediaURL)
				}
				if seenKW[kw] {
					kw = fmt.Sprintf("%s %s", kw, c.Handle)
				}
				seenKW[kw] = true
				img := searxImage(kw, seenImg)
				if img == "" {
					c.MediaURL = ""
				} else {
					c.MediaURL = img
				}
			}
			botID, ok := botByHandle[c.Handle]
			if !ok {
				b := bots[rand.Intn(len(bots))]
				botID = b.ID
				c.Handle = b.Handle
			}
			var postBotID int64
			db.QueryRow(`SELECT bot_id FROM posts WHERE id=?`, c.PostID).Scan(&postBotID)
			if postBotID == 0 {
				c.PostID = sampled[rand.Intn(len(sampled))].ID
				db.QueryRow(`SELECT bot_id FROM posts WHERE id=?`, c.PostID).Scan(&postBotID)
			}
			if postBotID == botID {
				for _, p := range sampled {
					var pb int64
					db.QueryRow(`SELECT bot_id FROM posts WHERE id=?`, p.ID).Scan(&pb)
					if pb != botID {
						c.PostID = p.ID
						postBotID = pb
						break
					}
				}
				if postBotID == botID {
					for _, b := range bots {
						if b.ID != postBotID {
							botID = b.ID
							c.Handle = b.Handle
							break
						}
					}
				}
			}
			var parent any
			if c.ParentID != nil {
				var cnt int
				var parentBot int64
				db.QueryRow(`SELECT COUNT(*), bot_id FROM comments WHERE id=? AND post_id=?`, *c.ParentID, c.PostID).Scan(&cnt, &parentBot)
				if cnt > 0 && parentBot != botID {
					parent = *c.ParentID
				} else {
					// invalid parent (wrong post or self-reply) -> drop to top-level, don't force random
					parent = nil
				}
			}
			created := time.Now().Add(-time.Duration(rand.Intn(3*24))*time.Hour).Add(-time.Duration(rand.Intn(60)) * time.Minute)
			_, err := db.Exec(`INSERT INTO comments(post_id, bot_id, parent_comment_id, content, media_url, created_at) VALUES(?,?,?,?,?,?)`,
				c.PostID, botID, parent, c.Content, c.MediaURL, created.Format(time.RFC3339))
			if err != nil {
				fmt.Printf("comment fail %v\n", err)
				continue
			}
			done++
			fmt.Printf("comment %d/%d @%s on post %d parent %v %.30s media:%v\n", done, n, c.Handle, c.PostID, parent, c.Content, c.MediaURL != "")
		}
	}
	return nil
}

func getKeys(m map[string]bool) []string {
	var k []string
	for s := range m {
		k = append(k, s)
		if len(k) > 20 {
			break
		}
	}
	return k
}

func weightedPick[T any](a []T, n int) []T {
	if n >= len(a) {
		return a
	}
	weights := make([]float64, len(a))
	var sum float64
	for i := range a {
		w := 1.0 / float64(i+1)
		weights[i] = w
		sum += w
	}
	var out []T
	used := map[int]bool{}
	for len(out) < n {
		r := rand.Float64() * sum
		var acc float64
		for i, w := range weights {
			acc += w
			if r < acc && !used[i] {
				out = append(out, a[i])
				used[i] = true
				break
			}
		}
		if len(used) == len(a) {
			break
		}
	}
	return out
}

func pick[T any](a []T, n int) []T {
	if n >= len(a) {
		return a
	}
	rand.Shuffle(len(a), func(i, j int) { a[i], a[j] = a[j], a[i] })
	return a[:n]
}
