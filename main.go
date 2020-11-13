package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mysftp"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var router *chi.Mux

func init() {
	router = chi.NewRouter()
	router.Use(middleware.Recoverer)

}

func routers() *chi.Mux {
	router.Get("/", ping)
	router.Post("/newConn", testPost)
	//router.Get("/download", Get)
	router.Put("/upload", upload)

	return router
}
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func ping(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Pong"})
}

func logger() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(time.Now(), r.Method, r.URL)
		router.ServeHTTP(w, r) // dispatch the request
	})
}

type Post struct {
	Host     string `json:"host"`
	User     string `json:"user"`
	Password string `json:"password"`
	Port     int    `json:"port"`
}

var post Post

var mysftpTemp *mysftp.SftpClient
var errs error

func testPost(w http.ResponseWriter, req *http.Request) {

	err := json.NewDecoder(req.Body).Decode(&post)
	if err != nil {
		log.Printf("error decoding sakura response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		log.Printf("sakura response: %q", post.Host)

	}
	log.Println(post.Host)
	mysftpTemp, errs = mysftp.NewConn(post.Host, post.User, post.Password, post.Port)

}

func upload(w http.ResponseWriter, req *http.Request) {

	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	req.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := req.FormFile("remoteFile")
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our mysftpTemp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("./tempFolder", handler.Filename)
	fmt.Printf("tempFile: %+v\n", tempFile.Name())
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)

	if err != nil {
		fmt.Println(err)
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)

	mysftpTemp.Put(tempFile.Name(), handler.Filename)

}

func main() {
	routers()
	fmt.Println("Listening")
	http.ListenAndServe(":8100", logger())
}
