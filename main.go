package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/skip2/go-qrcode"

	"github.com/gorilla/mux"
)

func generateManifest(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		showInternalError(err, w)
		return
	}

	ipa := r.Form.Get("ipa")
	identifier := r.Form.Get("identifier")
	version := r.Form.Get("version")
	title := r.Form.Get("title")

	manifest := manifestContent(ipa, identifier, version, title)
	const name = "manifest.plist"

	w.Header().Add("Content-Disposition", "Attachment; filename=manifest.plist")
	http.ServeContent(w, r, name, time.Now(), manifest)
}

func showInternalError(err error, w http.ResponseWriter) {
	fmt.Println(fmt.Errorf("Error: %v", err))
	w.WriteHeader(http.StatusInternalServerError)
}

func createQRCode(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()

	if err != nil {
		showInternalError(err, w)
		return
	}

	manifest := r.Form.Get("manifest")
	build := fmt.Sprintf("itms-services://?action=download-manifest&amp;url=%s", manifest)

	var png []byte
	png, errQRCode := qrcode.Encode(build, qrcode.Medium, 256)

	if errQRCode != nil {
		showInternalError(errQRCode, w)
		return
	}

	imgBase64Str := base64.StdEncoding.EncodeToString(png)

	img2html := "<html><body><img src=\"data:image/png;base64," + imgBase64Str + "\" /></body></html>"

	w.Write([]byte(fmt.Sprintf(img2html)))
}

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/generate", generateManifest).Methods("POST")
	r.HandleFunc("/qrcode", createQRCode).Methods("POST")
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("./assets/index.html")
		if err != nil {
			log.Print("template parsing error: ", err)
		}
		err = t.Execute(w, nil)
		if err != nil {
			log.Print("template executing error: ", err)
		}
	}).Methods("GET")

	return r
}

func getPort() string {
	p := os.Getenv("PORT")
	if p != "" {
		return ":" + p
	}
	return ":8080"
}

func main() {
	r := newRouter()
	http.ListenAndServe(getPort(), r)
}

func manifestContent(ipa string, identifier string, version string, title string) io.ReadSeeker {

	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
	<plist version="1.0">
	<dict>
		<key>items</key>
		<array>
			<dict>
				<key>assets</key>
				<array>
					<dict>
						<key>kind</key>
						<string>software-package</string>
						<key>url</key>
						<string>%s</string>
					</dict>
				</array>
				<key>metadata</key>
				<dict>
					<key>bundle-identifier</key>
					<string>%s</string>
					<key>bundle-version</key>
					<string>%s</string>
					<key>kind</key>
					<string>software</string>
					<key>title</key>
					<string>%s</string>
				</dict>
			</dict>
		</array>
	</dict>
	</plist>
	`, ipa, identifier, version, title)

	return bytes.NewReader([]byte(content))
}
