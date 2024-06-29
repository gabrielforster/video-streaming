package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	FILE_PATH = "uploads"
)

var port = flag.String("port", ":8087", "port which the filer server is going to listen")

func main() {
	flag.Parse()

	err := os.MkdirAll(FILE_PATH, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/upload", uploadHandler)

	fmt.Println("starting filer server")
	err = http.ListenAndServe(*port, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20) // 100 MB files
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	nowTimestamp := fmt.Sprint(time.Now().Unix())
	filename := nowTimestamp + r.Header.Get("filename")
	fileDst, err := os.Create(FILE_PATH + "/" + filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer fileDst.Close()

	// TODO: change from fs write to bucket upload
	if _, err := io.Copy(fileDst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go func() {
		postData := strings.NewReader(fmt.Sprintf(`{"filename": "%s"}`, filename))

		_, err := http.Post(
			"http://localhost:8088/process",
			"application/json",
			postData,
		)
		if err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Fprintf(w, "success\n")
}
