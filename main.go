package main

import (
	"html/template"
	"log"
	"net/http"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

type Card struct {
	Title string
	Body  string
}

type HomeData struct {
	Cards []Card
}

func main() {
	// Static files (optional)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/clicked", clickHandler)

	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data := HomeData{
		Cards: []Card{
			{Title: "Card One", Body: "This is the first reusable card component."},
			{Title: "Card Two", Body: "Components are just named templates in Go."},
		},
	}
	err := templates.ExecuteTemplate(w, "base.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func clickHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<p>Button clicked via htmx!</p>"))
}
