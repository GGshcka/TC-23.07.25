package main

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ServerData struct {
	Message  string
	Port     int
	Server   *http.Server
	Template *template.Template
	Items    [3]string
}

func NewServer(message string, port int, readTimeoutInSec int, writeTimeoutInSec int) ServerData {

	sd := ServerData{
		Message:  message,
		Template: template.Must(template.ParseGlob("templates/*.html")),
		Port:     port,
		Items:    [3]string{"", "", ""},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", sd.Handler) // <-- Привязка к методу структуры

	sd.Server = &http.Server{
		Addr:         ":" + strconv.Itoa(sd.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(readTimeoutInSec) * time.Second,
		WriteTimeout: time.Duration(writeTimeoutInSec) * time.Second,
	}

	return sd
}

func (sd ServerData) StartServer() {
	fmt.Println("Starting server at port " + strconv.Itoa(sd.Port))
	err := sd.ClearFolder("downloaded/")
	if err != nil {
		fmt.Println("Error removing the folder:", err)
	}
	err = sd.ClearFolder("archives/")
	if err != nil {
		fmt.Println("Error removing the folder:", err)
	}
	err = sd.Server.ListenAndServe()
	if err != nil {
		fmt.Println("Error starting the server:", err)
	}
}

func (sd *ServerData) Handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html")
		err := sd.Template.ExecuteTemplate(w, "main.html", sd)
		if err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		break
	case http.MethodPost:
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
			return
		}
		action := r.FormValue("action") // получать значение pressed button
		fmt.Println("Action:", action)
		switch action {
		case "upload":
			taskQueue <- Task{
				Name: "upload",
				Run:  func() { sd.FileGetFromSubmit(r) },
			}
			break
		case "clearFirst":
			taskQueue <- Task{
				Name: "clearFirst",
				Run: func() {
					err = os.Remove(sd.Items[0])
					if err != nil {
						fmt.Println(err)
					}
					err = os.Remove(strings.Replace(sd.Items[0], "downloaded", "archives", 1) + ".zip")
					if err != nil {
						fmt.Println(err)
					}
					sd.Items[0] = ""
				},
			}
			break
		case "clearSecond":
			taskQueue <- Task{
				Name: "clearSecond",
				Run: func() {
					err = os.Remove(sd.Items[1])
					if err != nil {
						fmt.Println(err)
					}
					err = os.Remove(strings.Replace(sd.Items[1], "downloaded", "archives", 1) + ".zip")
					if err != nil {
						fmt.Println(err)
					}
					sd.Items[1] = ""
				},
			}
			break
		case "clearThird":
			taskQueue <- Task{
				Name: "clearThird",
				Run: func() {
					err = os.Remove(sd.Items[2])
					if err != nil {
						fmt.Println(err)
					}
					err = os.Remove(strings.Replace(sd.Items[2], "downloaded", "archives", 1) + ".zip")
					if err != nil {
						fmt.Println(err)
					}
					sd.Items[2] = ""
				},
			}
			break
		case "downloadFirst":
			sd.Download(w, r, sd.Items[0])
			return
		case "downloadSecond":
			sd.Download(w, r, sd.Items[1])
			return
		case "downloadThird":
			sd.Download(w, r, sd.Items[2])
			return
		default:
			fmt.Println("Неизвестное действие")
			break
		}
		fmt.Println("Items now:", sd.Items)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		break
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		break
	}
}

func (sd *ServerData) FileGetFromSubmit(r *http.Request) {
	sd.Message = "‎"

	name := r.FormValue("linkInput")
	u, err := url.Parse(name)
	if err != nil {
		sd.Message = "Invalid URL"
		return
	}

	isValidExtension := false
	ext := filepath.Ext(u.Path)
	fmt.Println("File ext:", ext)
	for _, fileExt := range config.FileExtensions.Extensions {
		if fileExt.Value == ext {
			isValidExtension = true
		}
	}
	if !isValidExtension {
		sd.Message = "Invalid file type"
		return
	}

	err = os.MkdirAll("downloaded", os.ModePerm)
	if err != nil {
		fmt.Println("Can't create folder:", err)
		return
	}
	filename := "downloaded/" + time.Now().Format("02.01.06_15:04:05") + ext

	for i, v := range sd.Items {
		if v != "" && i == 2 {
			sd.Message = "Too many items!"
			return
		}
		if v == "" {
			sd.Items[i] = filename
			break
		}
	}

	resp, err := http.Get(name)
	if err != nil {
		sd.Message = "Template error: " + err.Error()
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		fmt.Println("Can't create file:", err)
		sd.Message = "Can't create file"
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Can't write into file:", err)
		sd.Message = "Can't write into file"
		return
	}
	io.Copy(os.Stdout, resp.Body)

	fmt.Println("File saved! Items:", sd.Items)

	sd.Archivate(filename)
	return
}

func (sd *ServerData) Archivate(v string) {
	err := os.MkdirAll("archives", os.ModePerm)
	if err != nil {
		fmt.Println("Can't create folder:", err)
		return
	}

	zipPath := strings.Replace(v, "downloaded", "archives", 1) + ".zip"

	zipFile, err := os.Create(zipPath)
	if err != nil {
		fmt.Println("Can't create file:", err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	file, err := os.Open(v)
	if err != nil {
		fmt.Println("Can't open file:", err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		fmt.Println("Can't get file info:", err)
		return
	}

	header, err := zip.FileInfoHeader(stat)
	if err != nil {
		fmt.Println("Can't create zip header:", err)
		return
	}

	header.Name = filepath.Base(v)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		fmt.Println("Can't create zip section:", err)
		return
	}

	_, err = io.Copy(writer, file)
	if err != nil {
		fmt.Println("Can't write into zip archive:", err)
		return
	}

	err = zipWriter.Close()
	if err != nil {
		fmt.Println("Can't close zip:", err)
		return
	}
}

func (sd *ServerData) Download(w http.ResponseWriter, r *http.Request, path string) {
	zipPath := strings.Replace(path, "downloaded", "archives", 1) + ".zip"

	file, err := os.Open(zipPath)
	if err != nil {
		fmt.Println("Can't open zip file:", err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		fmt.Println("Can't get file zip info:", err)
		return
	}

	fmt.Println("File size:", stat.Size(), "bytes")

	w.Header().Set("Content-Disposition", "attachment; filename="+stat.Name())
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, zipPath)
}

func (sd *ServerData) ClearFolder(folder string) error {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(folder, entry.Name())
		if !entry.IsDir() {
			err = os.Remove(path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
