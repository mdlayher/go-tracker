package api

import (
	"compress/gzip"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Error represents an error response from the API
type Error struct {
	Error string `json:"error"`
}

// Router handles the routing of HTTP API requests
func Router(w http.ResponseWriter, r *http.Request) {
	// API allows the following HTTP methods:
	//   - GET: read-only access to data
	//   - POST: create a new item via an API endpoint
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, ErrorResponse("Method not allowed"), 405)
		return
	}

	// Log API calls
	log.Printf("API: [http %s] %s %s\n", r.RemoteAddr, r.Method, r.URL.Path)

	// Split request path
	urlArr := strings.Split(r.URL.Path, "/")

	// Verify API method set
	if len(urlArr) < 3 {
		http.Error(w, ErrorResponse("No API call"), 404)
		return
	}

	// Check for an ID
	ID := -1
	if len(urlArr) == 4 {
		i, err := strconv.Atoi(urlArr[3])
		if err != nil || i < 1 {
			http.Error(w, ErrorResponse("Invalid integer ID"), 400)
			return
		}

		ID = i
	}

	// Response buffer, error
	res := []byte(ErrorResponse("Undefined API call: " + r.Method + " " + urlArr[2]))
	var err error

	// Choose API method
	switch urlArr[2] {
	// Files on tracker
	case "files":
		// GET
		if r.Method == "GET" {
			res, err = getFilesJSON(ID)
		}
	// Server status
	case "status":
		// GET
		if r.Method == "GET" {
			res, err = getStatusJSON()
		}
	// Users registered to tracker
	case "users":
		// GET
		if r.Method == "GET" {
			res, err = getUsersJSON(ID)
		}
	// Return error response
	default:
		http.Error(w, ErrorResponse("Undefined API call"), 404)
		return
	}

	// Check for errors, return 500
	if err != nil {
		http.Error(w, ErrorResponse("API could not generate response"), 500)
		return
	}

	// If requested, compress response using gzip
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Add("Content-Encoding", "gzip")

		// Write gzip'd response
		gz := gzip.NewWriter(w)
		if _, err := gz.Write(res); err != nil {
			log.Println(err.Error())
			return
		}

		if err := gz.Close(); err != nil {
			log.Println(err.Error())
		}

		return
	}

	if _, err := w.Write(res); err != nil {
		log.Println(err.Error())
	}

	return
}

// ErrorResponse returns an Error as JSON
func ErrorResponse(msg string) string {
	res := Error{
		msg,
	}

	out, err := json.Marshal(res)
	if err != nil {
		log.Println(err.Error())
		return `{"error":"` + msg + `"}`
	}

	return string(out)
}
