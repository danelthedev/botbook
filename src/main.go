package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func loadTemplates() *template.Template {
	t := template.New("").Funcs(template.FuncMap{
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
		"sub": func(a, b int) int { return a - b },
		"reactionTooltip": func(names []string) string {
			if len(names) == 0 {
				return ""
			}
			preview := names
			if len(names) > 3 {
				preview = names[:3]
			}
			result := strings.Join(preview, ", ")
			if len(names) > 3 {
				result += fmt.Sprintf(" +%d more", len(names)-3)
			}
			return result
		},
		"formatTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 - 3:04 PM")
		},
	})
	t = template.Must(t.ParseGlob("templates/*.html"))
	t = template.Must(t.ParseGlob("templates/components/*.html"))
	return t
}

var templates = loadTemplates()

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/feed", feedHandler)
	http.HandleFunc("/posts/{id}/reactions", postReactionsHandler)
	http.HandleFunc("/posts/{id}/comments", postCommentsHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
