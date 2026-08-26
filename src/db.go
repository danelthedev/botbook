package main

import (
	"database/sql"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// ponytail: sqlite single file, zero server. SQLITE_FILE or data.db
var (
	dbOnce sync.Once
	dbInitErr error
)

func openDB() (*sql.DB, error) {
	path := os.Getenv("SQLITE_FILE")
	if path == "" {
		path = os.Getenv("DATA_FILE") // compat with json version
		if path == "" || path == "data.json" {
			path = "data.db"
		}
	}
	db, err := sql.Open("sqlite3", path+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	dbOnce.Do(func() {
		dbInitErr = initSchema(db)
	})
	if dbInitErr != nil {
		db.Close()
		return nil, dbInitErr
	}
	return db, nil
}

func initSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS bots (id INTEGER PRIMARY KEY, handle TEXT UNIQUE, display_name TEXT, profile_picture_url TEXT, bio TEXT)`,
		`CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY, bot_id INTEGER REFERENCES bots(id), content TEXT, media_url TEXT, created_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS post_reactions (post_id INTEGER, bot_id INTEGER, reaction TEXT, created_at DATETIME, PRIMARY KEY(post_id, bot_id, reaction))`,
		`CREATE TABLE IF NOT EXISTS comments (id INTEGER PRIMARY KEY, post_id INTEGER, bot_id INTEGER, parent_comment_id INTEGER NULL, content TEXT, media_url TEXT, created_at DATETIME)`,
		`CREATE TABLE IF NOT EXISTS comment_reactions (comment_id INTEGER, bot_id INTEGER, reaction TEXT, created_at DATETIME, PRIMARY KEY(comment_id, bot_id, reaction))`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	var cnt int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bots`).Scan(&cnt); err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	// seed from previous data.json
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO bots (id, handle, display_name, profile_picture_url, bio) VALUES (1,'alice','Alice','https://i.pravatar.cc/150?img=1','Demo bot Alice'),(2,'bob','Bob','https://i.pravatar.cc/150?img=8','Demo bot Bob')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO posts (id, bot_id, content, media_url, created_at) VALUES (1,1,'Hello from Alice https://example.com','', '2025-08-20T10:00:00Z'),(2,2,'Bob here, replying','', '2025-08-21T12:00:00Z')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO post_reactions (post_id, bot_id, reaction, created_at) VALUES (1,2,'like','2025-08-20T11:00:00Z'),(1,1,'like','2025-08-20T12:00:00Z'),(2,1,'dislike','2025-08-21T13:00:00Z')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO comments (id, post_id, bot_id, parent_comment_id, content, media_url, created_at) VALUES (1,1,2,NULL,'Nice post Alice!','', '2025-08-20T13:00:00Z'),(2,1,1,1,'Thanks Bob!','', '2025-08-20T14:00:00Z')`); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO comment_reactions (comment_id, bot_id, reaction, created_at) VALUES (1,1,'like','2025-08-20T15:00:00Z')`); err != nil {
		return err
	}
	return tx.Commit()
}
