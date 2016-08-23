// Demo of Google Vision API on App Engine
package main

import (
	"encoding/base64"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	vision "google.golang.org/api/vision/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"html/template"
	"io/ioutil"
	"net/http"
)

// HTML Template for the home page
var homeTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

// Main init function to assign paths to handlers
func init() {

	// Home page (& catch-all)
	http.HandleFunc("/", HomeHandler)

	// API to upload file to
	http.HandleFunc("/upload", UploadFileHandler)

}

// Home page handler
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, ">>> Home Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	// Execute the home page template
	if err := homeTemplate.Execute(w, template.FuncMap{
		"Version": appengine.VersionID(c),
	}); err != nil {
		log.Errorf(c, "Error with homeTemplate: %v", err)
		http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// API to upload file to
func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, ">>> Upload File Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	// Parse multipart form in a 32 Mb buffer
	r.ParseMultipartForm(32 << 20)

	// Extract file (code assumes single file, i.e. no "multiple" in the HTML form)
	file, _, err := r.FormFile("select_files")
	if err != nil {
		log.Errorf(c, "Error getting : %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Read data from file
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf(c, "Error reading file : %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode data in []byte to Base 64 in string
	bodyString := base64.StdEncoding.EncodeToString(data)

	// Create Service Account Client with relevant scopes
	serviceAccountClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(c,
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/cloud-platform"),
			Base: &urlfetch.Transport{
				Context: c,
			},
		},
	}

	// Create Vision API Service using Service Account Client
	srv, err := vision.New(serviceAccountClient)
	if err != nil {
		log.Errorf(c, "Error, getting service account: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a Vision API request
	req := &vision.AnnotateImageRequest{
		// Apply image which is encoded by base64.
		Image: &vision.Image{
			Content: bodyString,
		},
		// Apply features to indicate what type of image detection.
		Features: []*vision.Feature{
			{
				MaxResults: 10,
				Type:       "LABEL_DETECTION",
			},
			{
				MaxResults: 10,
				Type:       "LANDMARK_DETECTION",
			},
		},
	}

	// Create single request batch
	batch := &vision.BatchAnnotateImagesRequest{
		Requests: []*vision.AnnotateImageRequest{req},
	}

	// Execute the "vision.images.annotate".
	res, err := srv.Images.Annotate(batch).Do()
	if err != nil {
		log.Errorf(c, "Error, executing annotae: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON to API/user
	WriteJSON(w, &res)

}
