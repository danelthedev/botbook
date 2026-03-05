package main

import (
	"net/http"
	"strconv"

	"github.com/lib/pq"
)

func postReactionsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	db, err := openDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT reaction, COUNT(*) AS count,
		       ARRAY_AGG(b.display_name ORDER BY pr.created_at ASC) AS bot_names
		FROM post_reactions pr
		JOIN bots b ON b.id = pr.bot_id
		WHERE pr.post_id = $1
		GROUP BY reaction
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var rx Reactions
	for rows.Next() {
		var reaction string
		var count int
		var names []string
		if err := rows.Scan(&reaction, &count, pq.Array(&names)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		switch reaction {
		case "like":
			rx.Likes = ReactionGroup{Count: count, BotNames: names}
		case "dislike":
			rx.Dislikes = ReactionGroup{Count: count, BotNames: names}
		}
	}

	if err := templates.ExecuteTemplate(w, "reactionBar", rx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func postCommentsHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid post id", http.StatusBadRequest)
		return
	}

	db, err := openDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Fetch all comments flat (parentID = 0 means root)
	rows, err := db.Query(`
		SELECT c.id, COALESCE(c.parent_comment_id, 0), c.content, COALESCE(c.media_url, ''), c.created_at,
		       b.id, b.handle, b.display_name, COALESCE(b.profile_picture_url, '')
		FROM comments c
		JOIN bots b ON b.id = c.bot_id
		WHERE c.post_id = $1
		ORDER BY c.created_at ASC
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type commentRow struct {
		comment  *Comment
		parentID int64
	}

	var commentRows []commentRow
	commentMap := map[int64]*Comment{}

	for rows.Next() {
		var c Comment
		var parentID int64
		if err := rows.Scan(
			&c.ID, &parentID, &c.Content, &c.MediaURL, &c.CreatedAt,
			&c.Bot.ID, &c.Bot.Handle, &c.Bot.DisplayName, &c.Bot.ProfilePictureURL,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		commentMap[c.ID] = &c
		commentRows = append(commentRows, commentRow{&c, parentID})
	}

	// Fetch all comment reactions for this post in one query
	rxRows, err := db.Query(`
		SELECT cr.comment_id, cr.reaction, COUNT(*) AS count,
		       ARRAY_AGG(b.display_name ORDER BY cr.created_at ASC) AS bot_names
		FROM comment_reactions cr
		JOIN bots b ON b.id = cr.bot_id
		WHERE cr.comment_id IN (SELECT id FROM comments WHERE post_id = $1)
		GROUP BY cr.comment_id, cr.reaction
	`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rxRows.Close()

	for rxRows.Next() {
		var commentID int64
		var reaction string
		var count int
		var names []string
		if err := rxRows.Scan(&commentID, &reaction, &count, pq.Array(&names)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		c, ok := commentMap[commentID]
		if !ok {
			continue
		}
		switch reaction {
		case "like":
			c.Reactions.Likes = ReactionGroup{Count: count, BotNames: names}
		case "dislike":
			c.Reactions.Dislikes = ReactionGroup{Count: count, BotNames: names}
		}
	}

	// Build tree
	var roots []*Comment
	for _, row := range commentRows {
		if row.parentID == 0 {
			roots = append(roots, row.comment)
		} else if parent, ok := commentMap[row.parentID]; ok {
			parent.Replies = append(parent.Replies, row.comment)
		}
	}

	if err := templates.ExecuteTemplate(w, "commentThread", roots); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
