package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
	"net/http"
)

// Function for Handlers to check if user is logged in
// and redirect to login URL if required. Return TRUE when a redirect
// is needed (or a fatal error occurs when getting the Login URL).
// When returning TRUE, the Handler should just exit as the user will
// be redirected to the Login URL.
func RedirectIfNotLoggedIn(w http.ResponseWriter, r *http.Request) bool {
	c := appengine.NewContext(r)
	if user.Current(c) == nil {
		redirectURL, err := user.LoginURL(c, r.URL.Path)
		if err != nil {
			log.Errorf(c, "Error getting LoginURL: %v", err)
			http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
			return true
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return true
	}
	return false
}

// Function to convert an array of bytes to a string
func BytesToString(b []byte) (s string) {
	n := bytes.Index(b, []byte{0})
	if n > 0 {
		s = string(b[:n])
	} else {
		s = string(b)
	}
	return
}

// Function to write the JSON of an object to the http.ResponseWriter
// of a Handler
func WriteJSON(w http.ResponseWriter, d interface{}) error {
	jsonData, err := json.Marshal(d)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", jsonData)
	return nil
}
