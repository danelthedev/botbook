package llm_interaction

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

type genBot struct {
	Handle            string `json:"handle"`
	DisplayName       string `json:"display_name"`
	Bio               string `json:"bio"`
	ProfilePictureURL string `json:"profile_picture_url"`
}

var disclosureRe = regexp.MustCompile(`(?i)\s*[\|·\-—]*\s*(parody(\s*account)?|fictional(\s*character)?|not a real person|not real|bot\s*\|\s*parody|\(parody\)|parody\s*bot)\s*[\|·\-—]*\s*`)

func stripDisclosure(s string) string {
	s = disclosureRe.ReplaceAllString(s, " ")
	s = regexp.MustCompile(`\s{2,}`).ReplaceAllString(s, " ")
	s = strings.Trim(s, " |·-—")
	return strings.TrimSpace(s)
}

func CreateBots(db *sql.DB, n int) error {
	if n <= 0 {
		return nil
	}
	prompt := fmt.Sprintf(`Generate %d diverse characters for a social media simulation. Spread: normies, trolls, meme pages, gamers, manosphere, plus others (tech bro, fitness, spiritual, conspiracy, etc). Each is a caricature.

Return JSON array only, no extra text:
[{"handle":"lowercase 3-12 alphanumeric","display_name":"...","bio":"max 160 chars, personality only","profile_picture_url":""}]

Handles unique, lowercase, no @. Bio reflects personality only. Leave profile_picture_url empty.`, n)

	raw, err := callLLM(prompt)
	if err != nil {
		return err
	}
	js := extractJSON(raw)
	var bots []genBot
	if err := json.Unmarshal([]byte(js), &bots); err != nil {
		return fmt.Errorf("bots json %v raw %s", err, raw)
	}
	if len(bots) == 0 {
		return fmt.Errorf("no bots returned")
	}
	for i, b := range bots {
		if i >= n {
			break
		}
		b.Handle = strings.ToLower(strings.TrimSpace(b.Handle))
		b.Handle = strings.TrimPrefix(b.Handle, "@")
		b.Handle = strings.ReplaceAll(b.Handle, " ", "")
		if b.Handle == "" {
			b.Handle = fmt.Sprintf("bot%d", i)
		}
		var cnt int
		db.QueryRow(`SELECT COUNT(*) FROM bots WHERE handle=?`, b.Handle).Scan(&cnt)
		if cnt > 0 {
			b.Handle = fmt.Sprintf("%s%d", b.Handle, i+100)
		}
		b.Bio = stripDisclosure(b.Bio)
		if len([]rune(b.Bio)) > 160 {
			b.Bio = string([]rune(b.Bio)[:160])
		}
		pic := strings.TrimSpace(b.ProfilePictureURL)
		if pic == "" || !strings.HasPrefix(pic, "http") {
			h := fnv.New32a()
			h.Write([]byte(b.Handle + b.Bio))
			v := h.Sum32()
			if v%2 == 0 {
				pic = fmt.Sprintf("https://i.pravatar.cc/300?u=%s-%d", b.Handle, v%10000)
			} else {
				gender := "men"
				if v%3 == 0 {
					gender = "women"
				}
				num := v%70 + 1
				pic = fmt.Sprintf("https://randomuser.me/api/portraits/%s/%d.jpg", gender, num)
			}
		}
		_, err := db.Exec(`INSERT INTO bots(handle, display_name, profile_picture_url, bio) VALUES(?,?,?,?)`,
			b.Handle, b.DisplayName, pic, b.Bio)
		if err != nil {
			fmt.Printf("bot insert skip %s: %v\n", b.Handle, err)
			continue
		}
		fmt.Printf("bot +%s (%s)\n", b.Handle, b.DisplayName)
	}
	return nil
}

func CleanExistingBios(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, bio FROM bots`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var bio string
		rows.Scan(&id, &bio)
		nbio := stripDisclosure(bio)
		if nbio != bio {
			db.Exec(`UPDATE bots SET bio=? WHERE id=?`, nbio, id)
			fmt.Printf("clean bio %d\n", id)
		}
	}
	return nil
}
