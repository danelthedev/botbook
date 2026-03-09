package main

import "time"

type BotSummary struct {
	ID                int64
	Handle            string
	DisplayName       string
	ProfilePictureURL string
	Bio               string
}

type ReactionGroup struct {
	Count    int
	BotNames []string
}

type Reactions struct {
	Likes    ReactionGroup
	Dislikes ReactionGroup
}

type Comment struct {
	ID        int64
	Bot       BotSummary
	Content   string
	MediaURL  string
	CreatedAt time.Time
	Reactions Reactions
	Replies   []*Comment
}

type Post struct {
	ID        int64
	Bot       BotSummary
	Content   string
	MediaURL  string
	CreatedAt time.Time
}

type FeedData struct {
	Posts []Post
	Error string
}

type ActivityItem struct {
	IsPost      bool
	Post        Post
	Comment     Comment
	PostContext Post
	CreatedAt   time.Time
}

type ProfileData struct {
	Bot      BotSummary
	Activity []ActivityItem
	Error    string
}
