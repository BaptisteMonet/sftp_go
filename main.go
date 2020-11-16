package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mysftp"
	"net/http"
	"os"
	"strconv"
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
	router.Get("/download", download)
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
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	req.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	remoteFile, remoteFileHeader, remoteFileErr := req.FormFile("remoteFile")
	if remoteFileErr != nil {
		fmt.Println("Error Retrieving the File", remoteFileErr)
		return
	}
	defer remoteFile.Close()
	createTemporaryFile, createTemporaryFileErr := ioutil.TempFile("./uploadFolder", remoteFileHeader.Filename)
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

func download(w http.ResponseWriter, req *http.Request) {
	os.Chdir("./downloadFolder")
	req.ParseForm()
	remoteFileName := req.FormValue("remoteFileName")
	mySftpDataConnection.Get(remoteFileName, remoteFileName)
	attachment := fmt.Sprintf("attachment; filename=" + strconv.Quote(remoteFileName))
	w.Header().Set("Content-Disposition", attachment)
	w.Header().Set("Content-Type", req.Header.Get("Content-Type"))
	openLocalFileUploaded, openLocalFileUploadedErr := os.Open(remoteFileName)
	if openLocalFileUploadedErr != nil {
		log.Println("openLocalFileUploadedErr open before send dl=> ", openLocalFileUploadedErr)
		return
	}
	io.Copy(w, openLocalFileUploaded)
	defer os.Remove(remoteFileName) // deleteFile uploaded on server before transfer to sftp
}

func main() {
	routers()
	fmt.Println("Listening")
	http.ListenAndServe(":8100", logger())
}
