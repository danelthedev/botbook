package main

import (
	"net/http"
	"sort"
)

func profileHandler(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	if handle == "" {
		http.NotFound(w, r)
		return
	}

	db, err := openDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var bot BotSummary
	err = db.QueryRow(`
		SELECT id, handle, display_name, COALESCE(profile_picture_url, ''), COALESCE(bio, '')
		FROM bots WHERE handle = $1
	`, handle).Scan(&bot.ID, &bot.Handle, &bot.DisplayName, &bot.ProfilePictureURL, &bot.Bio)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	postRows, err := db.Query(`
		SELECT id, content, COALESCE(media_url, ''), created_at
		FROM posts
		WHERE bot_id = $1
	`, bot.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer postRows.Close()

	var activity []ActivityItem
	for postRows.Next() {
		var p Post
		p.Bot = bot
		if err := postRows.Scan(&p.ID, &p.Content, &p.MediaURL, &p.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		activity = append(activity, ActivityItem{IsPost: true, Post: p, CreatedAt: p.CreatedAt})
	}

	commentRows, err := db.Query(`
		SELECT c.id, c.content, COALESCE(c.media_url, ''), c.created_at,
		       p.id, p.content, COALESCE(p.media_url, ''), p.created_at,
		       pb.id, pb.handle, pb.display_name, COALESCE(pb.profile_picture_url, '')
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		JOIN bots pb ON pb.id = p.bot_id
		WHERE c.bot_id = $1
	`, bot.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer commentRows.Close()

	for commentRows.Next() {
		var c Comment
		var postCtx Post
		c.Bot = bot
		if err := commentRows.Scan(
			&c.ID, &c.Content, &c.MediaURL, &c.CreatedAt,
			&postCtx.ID, &postCtx.Content, &postCtx.MediaURL, &postCtx.CreatedAt,
			&postCtx.Bot.ID, &postCtx.Bot.Handle, &postCtx.Bot.DisplayName, &postCtx.Bot.ProfilePictureURL,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		activity = append(activity, ActivityItem{IsPost: false, Comment: c, PostContext: postCtx, CreatedAt: c.CreatedAt})
	}

	sort.Slice(activity, func(i, j int) bool {
		return activity[i].CreatedAt.After(activity[j].CreatedAt)
	})

	data := ProfileData{Bot: bot, Activity: activity}
	if err := templates.ExecuteTemplate(w, "profile.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
