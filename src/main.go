package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var templates = template.Must(
	template.New("").Funcs(template.FuncMap{
		"dict": func(pairs ...any) (map[string]any, error) {
			if len(pairs)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs); i += 2 {
				key, ok := pairs[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				m[key] = pairs[i+1]
			}
			return m, nil
		},
		"add": func(a, b int) int { return a + b },
	}).ParseGlob("templates/*.html"),
)

type HomeData struct {
	Tables []string
	Error  string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

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

	err = templates.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
