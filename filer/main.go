package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

const (
	FILE_PATH   = "uploads"
	BUCKET_NAME = "filer-videos"
)

type FileMetadata struct {
	Id       string `json:"id"`
	Filename string `json:"filename"`
}

var cfg = aws.Config{
	Endpoint:         aws.String("http://localhost:4566"),
	Region:           aws.String("sa-east-1"),
	DisableSSL:       aws.Bool(true),
	S3ForcePathStyle: aws.Bool(true),
}

var port = flag.String("port", ":8087", "port which the filer server is going to listen")

func main() {
	flag.Parse()

	sess := session.Must(session.NewSession(&cfg))
	s3Client := s3.New(sess)

	err := os.MkdirAll(FILE_PATH, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}

	http.HandleFunc("/upload", uploadHandler(s3Client))
	http.HandleFunc("/signed", retrieveSignedURLHandler(s3Client))

	fmt.Println("starting filer server")
	err = http.ListenAndServe(*port, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func retrieveSignedURLHandler(s3Client *s3.S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		filename := r.Header.Get("filename")
		if filename == "" {
			http.Error(w, "filename header is required", http.StatusBadRequest)
			return
		}

		req, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
			Bucket: aws.String(BUCKET_NAME),
			Key:    aws.String(filename),
		})

		url, headers, err := req.PresignRequest(15 * time.Minute)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println("signed url:", url)
		fmt.Println("headers:", headers)
		for key, value := range headers {
			w.Header().Set(key, value[0])
		}

		w.Write([]byte(url))
	}
}

func uploadHandler(s3Client *s3.S3) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		filename := r.Header.Get("filename")
		fileMetadata := FileMetadata{
			Id:       uuid.New().String(),
			Filename: filename,
		}
        if fileMetadata.Filename == "" {
            fileMetadata.Filename = fileMetadata.Id
        }

        _, err = s3Client.PutObject(&s3.PutObjectInput{
            Bucket: aws.String(BUCKET_NAME),
            Key:    aws.String(fileMetadata.Filename),
            Body:   file,
        })
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }


		// TODO: move this a pub/sub (redis?)
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
}
