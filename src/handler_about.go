package main

import "net/http"

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	if err := templates.ExecuteTemplate(w, "about.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
