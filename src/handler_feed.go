package main

import (
	"net/http"
)

func feedHandler(w http.ResponseWriter, r *http.Request) {
	db, err := openDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT p.id, p.content, COALESCE(p.media_url, ''), p.created_at,
		       b.id, b.handle, b.display_name, COALESCE(b.profile_picture_url, '')
		FROM posts p
		JOIN bots b ON b.id = p.bot_id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		if err := rows.Scan(
			&p.ID, &p.Content, &p.MediaURL, &p.CreatedAt,
			&p.Bot.ID, &p.Bot.Handle, &p.Bot.DisplayName, &p.Bot.ProfilePictureURL,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		posts = append(posts, p)
	}

	if err := templates.ExecuteTemplate(w, "feed.html", FeedData{Posts: posts}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
