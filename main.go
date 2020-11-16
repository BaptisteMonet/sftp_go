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
	router.Get("/ping", ping)
	router.Post("/connect", connectToSftp)
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

var mySftpDataConnection *mysftp.SftpClient
var errs error

func connectToSftp(w http.ResponseWriter, req *http.Request) {

	newDecoderErr := json.NewDecoder(req.Body).Decode(&post)
	if newDecoderErr != nil {
		log.Printf("error decoding sakura response: %v", newDecoderErr)
		if e, ok := newDecoderErr.(*json.SyntaxError); ok {
			log.Printf("syntax error at byte offset %d", e.Offset)
		}
		log.Printf("sakura response: %q", post.Host)

	}
	log.Println(post.Host)
	mySftpDataConnection, errs = mysftp.CreateNewConnection(post.Host, post.User, post.Password, post.Port)

}

func upload(w http.ResponseWriter, req *http.Request) {

	fmt.Println("File Upload Endpoint Hit")

	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	req.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	remoteFile, remoteFileHeader, remoteFileErr := req.FormFile("remoteFile")
	if remoteFileErr != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(remoteFileErr)
		return
	}
	defer remoteFile.Close()
	fmt.Printf("Uploaded File: %+v\n", remoteFileHeader.Filename)
	fmt.Printf("File Size: %+v\n", remoteFileHeader.Size)
	fmt.Printf("MIME Header: %+v\n", remoteFileHeader.Header)

	createTemporaryFile, createTemporaryFileErr := ioutil.TempFile("./tempFolder", remoteFileHeader.Filename)
	fmt.Printf("createTemporaryFile: %+v\n", createTemporaryFile.Name())
	if createTemporaryFileErr != nil {
		fmt.Println(createTemporaryFileErr)
	}
	defer createTemporaryFile.Close()

	readUploadedFileAndConvertIntoByteArray, readUploadedErr := ioutil.ReadAll(remoteFile)

	if readUploadedErr != nil {
		fmt.Println(readUploadedErr)
	}

	createTemporaryFile.Write(readUploadedFileAndConvertIntoByteArray)

	mySftpDataConnection.Put(createTemporaryFile.Name(), remoteFileHeader.Filename)

}

func main() {
	routers()
	fmt.Println("Listening")
	http.ListenAndServe(":8100", logger())
}
