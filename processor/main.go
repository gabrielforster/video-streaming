package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
    PROCESSED_PATH = "processed/"
)
var port = flag.String("port", ":8088", "port which the processor server is going to listen")

func main() {
    flag.Parse()
    err := os.MkdirAll(PROCESSED_PATH, 0775)
    if err != nil {
        fmt.Println(err)
        return
    }

	http.HandleFunc("/process", processFile)

    fmt.Printf("processor server is running on port %s\n", *port)
    err = http.ListenAndServe(*port, nil)
    if err != nil {
        fmt.Println(err)
    }

}

type ProcessRequestBody struct {
	Filename string `json:"filename"`
}

func processFile(w http.ResponseWriter, r *http.Request) {
	var body ProcessRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	go processFileAsync(body.Filename)

	w.WriteHeader(http.StatusOK)
}

func processFileAsync(rawFilename string) {
    filename := strings.Split(rawFilename, ".")[0]
	fmt.Printf("processing file: %s\n", filename)

    inputFilePath := "../filer/uploads/" + rawFilename
    fmt.Printf("inputFilePath: %s\n", inputFilePath)
    outputPath := PROCESSED_PATH + filename + "/"
    fmt.Printf("outputPath: %s\n", outputPath)

    err := os.MkdirAll(outputPath, 0775)
    if err != nil {
        fmt.Println(err)
        return
    }

	cmd := exec.Command("ffmpeg",
		"-i", inputFilePath,
        "-f", "hls",
        "-hls_time", "10",
        "-hls_playlist_type", "vod",
        "-hls_flags", "independent_segments",
        "-hls_segment_type", "mpegts",
		"-hls_segment_filename", outputPath+"output_%03d.ts",
		outputPath+"output.m3u8",
	)

	fmt.Printf("starting processing file: %s\n", filename)
    output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
        fmt.Println(string(output))
        return
	}

	fmt.Printf("finished processing file: %s\n", filename)
}
