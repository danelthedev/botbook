package main

import (
	"database/sql"
	"net/http"
	"os"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data := HomeData{}

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		data.Error = err.Error()
		templates.ExecuteTemplate(w, "base.html", data)
		return
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`)
	if err != nil {
		data.Error = err.Error()
		templates.ExecuteTemplate(w, "base.html", data)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			data.Error = err.Error()
			templates.ExecuteTemplate(w, "base.html", data)
			return
		}
		data.Tables = append(data.Tables, name)
	}

	if err = templates.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
