package main

import (
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ApiResponse struct {
	Title string `json:title`
}

const getMetadataUrl = "https://playall.sharp-stream.com/getmetadata"

var dsn = os.Getenv("MAGNUM_DSN")
var formData = url.Values{
	"src": []string{"http://tx.sharp-stream.com/http_live.php?i=iasca.mp3"},
}

func main() {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{}
	lastPlayed := firstTitle(db)
	for {
		resp, err := client.PostForm(getMetadataUrl, formData)
		if err != nil {
			if strings.Contains(err.Error(), "TLS handshake timeout") {
				time.Sleep(time.Minute)
				continue
			}

			log.Fatal(err)
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		data := &ApiResponse{}
		err = json.Unmarshal(bodyBytes, data)
		if err != nil {
			log.Fatal(err)
		}

		if data.Title != lastPlayed {
			update(data.Title, db)
			lastPlayed = data.Title
		}

		time.Sleep(10 * time.Second)
	}
}

func firstTitle(db *sql.DB) string {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	var title string
	result, err := db.Query("SELECT title FROM plays ORDER BY id DESC LIMIT 1")
	defer result.Close()

	result.Next()
	err = result.Scan(&title)
	if err != nil {
		log.Println(err)
		return ""
	}

	log.Println("picking up from database: " + title)
	return title
}

func update(title string, db *sql.DB) {
	stmt, err := db.Prepare("INSERT INTO plays(title) VALUES (?)")
	defer stmt.Close()
	if err != nil {
		log.Fatal(err)
	}

	r, err := stmt.Exec(title)
	if err != nil {
		log.Fatal(err)
	}

	rows, _ := r.RowsAffected()
	if rows != 1 {
		log.Fatal("r.RowsAffected != 1")
	}

	log.Println(title)
}
