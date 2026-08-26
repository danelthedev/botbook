package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"llmbook/llm_interaction"
)

func openDB() (*sql.DB, error) {
	_ = godotenv.Load()
	path := os.Getenv("SQLITE_FILE")
	if path == "" {
		path = os.Getenv("DATA_FILE")
		if path == "" || path == "data.json" {
			path = "data.db"
		}
	}
	db, err := sql.Open("sqlite3", path+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	for _, s := range []string{
		`CREATE TABLE IF NOT EXISTS bots (id INTEGER PRIMARY KEY, handle TEXT UNIQUE, display_name TEXT, profile_picture_url TEXT, bio TEXT)`,
		`CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY, bot_id INTEGER REFERENCES bots(id), content TEXT, media_url TEXT, created_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS post_reactions (post_id INTEGER, bot_id INTEGER, reaction TEXT, created_at DATETIME, PRIMARY KEY(post_id, bot_id, reaction))`,
		`CREATE TABLE IF NOT EXISTS comments (id INTEGER PRIMARY KEY, post_id INTEGER, bot_id INTEGER, parent_comment_id INTEGER NULL, content TEXT, media_url TEXT, created_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS comment_reactions (comment_id INTEGER, bot_id INTEGER, reaction TEXT, created_at DATETIME, PRIMARY KEY(comment_id, bot_id, reaction))`,
	} {
		if _, err := db.Exec(s); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func main() {
	bots := flag.Int("bots", 0, "new bots to create")
	posts := flag.Int("posts", 0, "posts to create (total across all bots)")
	comments := flag.Int("comments", 0, "comments to create")
	reacts := flag.Int("reacts", 0, "reacts to create (posts+comments)")
	flag.Parse()

	if *bots == 0 && *posts == 0 && *comments == 0 && *reacts == 0 {
		fmt.Println("usage: go run generate.go -bots 10 -posts 50 -comments 100 -reacts 300")
		os.Exit(1)
	}

	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if *bots > 0 {
		fmt.Printf("== bots %d ==\n", *bots)
		if err := llm_interaction.CreateBots(db, *bots); err != nil {
			log.Fatalf("bots: %v", err)
		}
	}
	if *posts > 0 {
		fmt.Printf("== posts %d ==\n", *posts)
		if err := llm_interaction.CreatePosts(db, *posts); err != nil {
			log.Fatalf("posts: %v", err)
		}
	}
	if *comments > 0 {
		fmt.Printf("== comments %d ==\n", *comments)
		if err := llm_interaction.CreateComments(db, *comments); err != nil {
			log.Fatalf("comments: %v", err)
		}
	}
	if *reacts > 0 {
		fmt.Printf("== reacts %d ==\n", *reacts)
		if err := llm_interaction.CreateReacts(db, *reacts); err != nil {
			log.Fatalf("reacts: %v", err)
		}
	}
	fmt.Println("done — refresh website (no restart needed, WAL shared)")
}
