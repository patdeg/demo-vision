// my doc
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

func init() {

	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/upload", UploadFileHandler)

}

var homeTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

func HomeHandler(w http.ResponseWriter, r *http.Request) {

	c := appengine.NewContext(r)
	log.Debugf(c, "Home Handler")

	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	if err := homeTemplate.Execute(w, template.FuncMap{
		"Path":    r.URL.Path,
		"Version": appengine.VersionID(c),
	}); err != nil {
		log.Errorf(c, "Error with homeTemplate: %v", err)
		http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, "Upload File Handler")

	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	log.Debugf(c, "Form:%v ", r.Form)
	log.Debugf(c, "PostForm:%v ", r.PostForm)
	log.Debugf(c, "MultipartForm:%v ", r.MultipartForm)

	r.ParseMultipartForm(32 << 20)

	log.Debugf(c, "(2) Form:%v ", r.Form)
	log.Debugf(c, "(2) PostForm:%v ", r.PostForm)
	log.Debugf(c, "(2) MultipartForm:%v ", r.MultipartForm)

	file, handler, err := r.FormFile("select_files")
	if err != nil {
		log.Errorf(c, "Error getting : %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	log.Debugf(c, "Handler : %v", handler)
	log.Debugf(c, "Header : %v", handler.Header)

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Errorf(c, "Error reading file : %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Debugf(c, "Data : %v", BytesToString(data))

	bodyString := base64.StdEncoding.EncodeToString(data)
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

	srv, err := vision.New(serviceAccountClient)
	if err != nil {
		log.Errorf(c, "Error, getting service account: %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

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

	// To call multiple image annotation requests.
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

	WriteJSON(w, &res)
}
