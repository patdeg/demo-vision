// Demo of Google Vision API on App Engine
package main

import (
	"encoding/base64"
	"fmt"
	"github.com/mssola/user_agent"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	bigquery "google.golang.org/api/bigquery/v2"
	vision "google.golang.org/api/vision/v1"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// HTML Template for the home page
var homeTemplate = template.Must(template.New("index.html").Delims("[[", "]]").ParseFiles("index.html"))

// Main init function to assign paths to handlers
func init() {

	// Home page (& catch-all)
	http.HandleFunc("/", HomeHandler)

	// API to upload file to
	http.HandleFunc("/upload", UploadFileHandler)

	// Admin handler to create table in BigQuery
	http.HandleFunc("/init", CreateBigQueryTableHandler)

	// Redirect to BigQuery console at the right project
	http.HandleFunc("/bq", RedirectBigQueryConsoleHandler)

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
	file, header, err := r.FormFile("select_files")
	if err != nil {
		log.Errorf(c, "Error getting : %v", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Debugf(c, "Filename: %v", header.Filename)
	for k, v := range header.Header {
		log.Debugf(c, "%v: %v", k, v)
	}

	contentType := ""
	if len(header.Header["Content-Type"]) > 0 {
		contentType = header.Header["Content-Type"][0]
	}

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
				MaxResults: 5,
				Type:       "LANDMARK_DETECTION",
			},
			{
				MaxResults: 5,
				Type:       "LOGO_DETECTION",
			},
			{
				MaxResults: 1,
				Type:       "TEXT_DETECTION",
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
		log.Errorf(c, "Error, executing annotate: %v", err)
		log.Debugf(c, "Request: %v", ToJSON(req))
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	ua := user_agent.New(r.Header.Get("User-Agent"))
	engineName, engineversion := ua.Engine()
	browserName, browserVersion := ua.Browser()

	var bqLabels []bigquery.JsonValue

	if len(res.Responses) >= 0 {
		for _, l := range res.Responses[0].LabelAnnotations {
			bqLabel := map[string]bigquery.JsonValue{
				"Type":  "Label",
				"Label": l.Description,
				"Score": l.Score,
			}
			bqLabels = append(bqLabels, bqLabel)
		}
		for _, l := range res.Responses[0].LandmarkAnnotations {
			bqLabel := map[string]bigquery.JsonValue{
				"Type":  "Landmark",
				"Label": l.Description,
				"Score": l.Score,
			}
			bqLabels = append(bqLabels, bqLabel)
		}

		for _, l := range res.Responses[0].LogoAnnotations {
			bqLabel := map[string]bigquery.JsonValue{
				"Type":  "Logo",
				"Label": l.Description,
				"Score": l.Score,
			}
			bqLabels = append(bqLabels, bqLabel)
		}

		for _, l := range res.Responses[0].TextAnnotations {
			bqLabel := map[string]bigquery.JsonValue{
				"Type":  "Text",
				"Label": l.Description,
			}
			bqLabels = append(bqLabels, bqLabel)
			// Use only one text annotations (full text), ignore individual words
			break
		}

	}

	bq_req := &bigquery.TableDataInsertAllRequest{
		Kind: "bigquery#tableDataInsertAllRequest",
		Rows: []*bigquery.TableDataInsertAllRequestRows{
			{
				Json: map[string]bigquery.JsonValue{
					"User":           user.Current(c).ID,
					"Time":           time.Now(),
					"Labels":         bqLabels,
					"Filename":       header.Filename,
					"ContentType":    contentType,
					"Size":           len(data),
					"Country":        r.Header.Get("X-AppEngine-Country"),
					"Region":         r.Header.Get("X-AppEngine-Region"),
					"City":           r.Header.Get("X-AppEngine-City"),
					"IsMobile":       ua.Mobile(),
					"MozillaVersion": ua.Mozilla(),
					"Platform":       ua.Platform(),
					"OS":             ua.OS(),
					"EngineName":     engineName,
					"EngineVersion":  engineversion,
					"BrowserName":    browserName,
					"BrowserVersion": browserVersion,
					"UserAgent":      r.Header.Get("User-Agent"),
				},
			},
		},
	}

	err = StreamDataInBigquery(c, projectId, "demo", "vision", bq_req)
	if err != nil {
		log.Errorf(c, "Error while streaming visit to BigQuery: %v", err)
		log.Debugf(c, "Request: %v", ToJSON(bq_req))
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JSON to API/user
	WriteJSON(w, &res)

}

// Create BigQuery table in current project (admin only)
func CreateBigQueryTableHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, ">>> Create BigQuery Table Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	if user.IsAdmin(c) == false {
		log.Errorf(c, "Error, user %v is not authorized to create table in BigQuery", user.Current(c).Email)
		http.Error(w, "Unauthorized Access", http.StatusUnauthorized)
		return
	}

	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	newTable := &bigquery.Table{
		TableReference: &bigquery.TableReference{
			ProjectId: projectId,
			DatasetId: "demo",
			TableId:   "vision",
		},
		FriendlyName: "Vision API Demo Data",
		Schema: &bigquery.TableSchema{
			Fields: []*bigquery.TableFieldSchema{
				{Name: "User", Type: "STRING", Description: "User Id"},
				{Name: "Time", Type: "TIMESTAMP", Description: "Time"},
				{Name: "Labels", Type: "RECORD", Description: "Labels", Mode: "REPEATED",
					Fields: []*bigquery.TableFieldSchema{
						{Name: "Type", Type: "STRING", Description: "Label Type"},
						{Name: "Label", Type: "STRING", Description: "Label"},
						{Name: "Score", Type: "FLOAT", Description: "Score"},
					},
				},
				{Name: "Filename", Type: "STRING", Description: "Filename"},
				{Name: "ContentType", Type: "STRING", Description: "Content Type"},
				{Name: "Size", Type: "INTEGER", Description: "File size"},
				{Name: "Country", Type: "STRING", Description: "Country"},
				{Name: "Region", Type: "STRING", Description: "Region"},
				{Name: "City", Type: "STRING", Description: "City"},
				{Name: "IsMobile", Type: "BOOLEAN", Description: "IsMobile"},
				{Name: "MozillaVersion", Type: "STRING", Description: "MozillaVersion"},
				{Name: "Platform", Type: "STRING", Description: "Platform"},
				{Name: "OS", Type: "STRING", Description: "OS"},
				{Name: "EngineName", Type: "STRING", Description: "EngineName"},
				{Name: "EngineVersion", Type: "STRING", Description: "EngineVersion"},
				{Name: "BrowserName", Type: "STRING", Description: "BrowserName"},
				{Name: "BrowserVersion", Type: "STRING", Description: "BrowserVersion"},
				{Name: "UserAgent", Type: "STRING", Description: "UserAgent"},
			},
		},
	}

	err := CreateTableInBigQuery(c, newTable)
	if err != nil {
		log.Errorf(c, "Error requesting table creation in BigQuery: %v", err)
		http.Error(w, "Internal Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "<h1>Table Created</h1>")
}

// Redirect to BigQuery console at the right project
func RedirectBigQueryConsoleHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	log.Debugf(c, ">>> Redirect BigQuery Console Handler")

	// Check if user is logged in, otherwise exit (as redirect was requested)
	if RedirectIfNotLoggedIn(w, r) {
		return
	}

	if user.IsAdmin(c) == false {
		log.Errorf(c, "Error, user %v is not an admin", user.Current(c).Email)
		http.Error(w, "Unauthorized Access", http.StatusUnauthorized)
		return
	}

	// Get projectId from appspot.com URL
	projectId := strings.Replace(appengine.DefaultVersionHostname(c), ".appspot.com", "", 1)
	log.Debugf(c, "Project: %v", projectId)

	redirectURL := fmt.Sprintf("https://bigquery.cloud.google.com/welcome/%v", projectId)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
