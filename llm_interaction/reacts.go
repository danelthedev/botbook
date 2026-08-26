package llm_interaction

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

func CreateReacts(db *sql.DB, n int) error {
	if n <= 0 {
		return nil
	}
	botRows, _ := db.Query(`SELECT id FROM bots`)
	var bots []int64
	for botRows.Next() {
		var id int64
		botRows.Scan(&id)
		bots = append(bots, id)
	}
	botRows.Close()
	if len(bots) == 0 {
		return fmt.Errorf("no bots for reacts")
	}
	// load posts with recency
	type item struct{ ID int64; Time time.Time }
	var posts []item
	rows, _ := db.Query(`SELECT id, created_at FROM posts ORDER BY created_at DESC`)
	for rows.Next() {
		var id int64
		var ts string
		rows.Scan(&id, &ts)
		t, _ := time.Parse(time.RFC3339, ts)
		posts = append(posts, item{id, t})
	}
	rows.Close()
	var comments []item
	rows, _ = db.Query(`SELECT id, created_at FROM comments ORDER BY created_at DESC`)
	for rows.Next() {
		var id int64
		var ts string
		rows.Scan(&id, &ts)
		t, _ := time.Parse(time.RFC3339, ts)
		comments = append(comments, item{id, t})
	}
	rows.Close()

	choices := []string{"like", "dislike"}
	for i := 0; i < n; i++ {
		isPost := len(comments) == 0 || rand.Float32() < 0.6
		bot := bots[rand.Intn(len(bots))]
		reaction := choices[rand.Intn(2)]
		// recency bias: recent = more likely (3d window like comments)
		created := time.Now().Add(-time.Duration(rand.Intn(3*24))*time.Hour).Add(-time.Duration(rand.Intn(60)) * time.Minute).Format(time.RFC3339)
		if isPost && len(posts) > 0 {
			post := weightedItem(posts).ID
			_, err := db.Exec(`INSERT OR IGNORE INTO post_reactions(post_id, bot_id, reaction, created_at) VALUES(?,?,?,?)`, post, bot, reaction, created)
			if err != nil {
				fmt.Printf("react err %v\n", err)
			}
		} else if len(comments) > 0 {
			comment := weightedItem(comments).ID
			_, err := db.Exec(`INSERT OR IGNORE INTO comment_reactions(comment_id, bot_id, reaction, created_at) VALUES(?,?,?,?)`, comment, bot, reaction, created)
			if err != nil {
				fmt.Printf("react err %v\n", err)
			}
		}
		if i%50 == 0 {
			fmt.Printf("react %d/%d\n", i, n)
		}
	}
	fmt.Printf("reacts done %d\n", n)
	return nil
}

func weightedItem[T any](a []T) T {
	// assume a sorted DESC recent first, weight 1/(rank+1)
	// quick single pick
	n := len(a)
	weights := make([]float64, n)
	var sum float64
	for i := range a {
		w := 1.0 / float64(i+1)
		weights[i] = w
		sum += w
	}
	r := rand.Float64() * sum
	var acc float64
	for i, w := range weights {
		acc += w
		if r < acc {
			return a[i]
		}
	}
	return a[0]
}
