// Command server serves the standalone Career Quest demo frontend.
package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	root := os.DirFS("web")
	handler := http.FileServer(http.FS(root))

	log.Println("Career Quest demo: http://localhost:4173")
	log.Fatal(http.ListenAndServe(":4173", handler))
}
