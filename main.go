package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go-fuzzer/models"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// ffuf -u "$url" -w "wordlist.txt"

type FuzzTemplate struct {
	Id           int
	Url          string
	WordlistFile string
	OutputFile   string
	Ip           string
	WordCount    int
	Started      string
	Ended        string
	Finished     int
	Error        int
}

var DIR_WORDLISTS = "./wordlists/"
var DIR_FFUFOUTPUTS = "./ffuf_outputs/"
var DIR_MODELS = "./models/"
var DIR_TEMPLATES = "templates/*"

var DB_NAME = DIR_MODELS + "db.db"
var DB_ENGINE = "sqlite3"

var KEY_AUTH_HEADER = "Salahala"

var FLAG_HOSTCNTRL = false

func main() {
	var err error

	models.DB, err = sql.Open(DB_ENGINE, DB_NAME)
	if err != nil {
		log.Fatal("Cannnot connected to DB")
	}

	f, _ := os.Create("go-fuzzer.log")
	gin.DefaultWriter = io.MultiWriter(f, os.Stdout)

	route := gin.Default()

	route.LoadHTMLGlob(DIR_TEMPLATES)

	route.GET("/", index)
	route.POST("/", indexPost)
	route.GET("/index", index)
	route.POST("/index", indexPost)
	route.GET("/output/:id", output)

	route.Run("0.0.0.0" + ":" + "6169")
}

func index(c *gin.Context) {
	if FLAG_HOSTCNTRL && c.Request.Header[KEY_AUTH_HEADER] == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusNotFound,
			"error_message": "Not Found. :)",
		})
		return
	}

	fuzzs, err := models.GetFuzzs()
	if err != nil {
		fmt.Println(err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "I think somethings have just broken. :(",
		})
		return
	}
	fuzzTemps, err := returnFuzzTemps(fuzzs)
	if err != nil {
		fmt.Println(err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "I think somethings have just broken. :(",
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "go-fuzzer",
		"Fuzzs": fuzzTemps,
	})
}

func returnFuzzTemps(fuzzs []models.Fuzz) ([]FuzzTemplate, error) {
	fuzzTemps := make([]FuzzTemplate, len(fuzzs))
	for i, fuzz := range fuzzs {
		fuzzTemps[i].Id = fuzz.Id
		fuzzTemps[i].Url = fuzz.Url
		fuzzTemps[i].WordlistFile = fuzz.WordlistFile
		fuzzTemps[i].OutputFile = fuzz.OutputFile
		fuzzTemps[i].Ip = fuzz.Ip
		fuzzTemps[i].WordCount = fuzz.WordCount
		fuzzTemps[i].Started = time.UnixMilli(fuzz.Started).Format(time.RFC822)
		fuzzTemps[i].Ended = strconv.Itoa(int(fuzz.Ended))
		if fuzz.Ended != -1 {
			fuzzTemps[i].Ended = time.UnixMilli(fuzz.Ended).Format(time.RFC822)
		} else {
			if fuzz.Error != 1 {
				fuzzTemps[i].Ended = time.UnixMilli(fuzz.Started + int64(fuzz.WordCount)/20*1000).Format(time.RFC822)
			}
		}
		fuzzTemps[i].Finished = fuzzs[i].Finished
		fuzzTemps[i].Error = fuzzs[i].Error
	}
	return fuzzTemps, nil
}

func indexPost(c *gin.Context) {
	if FLAG_HOSTCNTRL && c.Request.Header[KEY_AUTH_HEADER] == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusNotFound,
			"error_message": "Not Found. :)",
		})
		return
	}
	url := c.PostForm("url")
	file, _ := c.FormFile("wordlist") //TODO file mevcutsa ve length'i aynıysa indirme?

	if strings.Contains(url, "FUZZ") == false {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "Girdiğin URL parametresinde FUZZ kelimesi yok. Nereye fuzz atacaz :)",
		})
		return
	}

	uploadedFilePath := DIR_WORDLISTS + file.Filename
	// Upload the file to specific dst.
	c.SaveUploadedFile(file, uploadedFilePath)

	go addFuzz(url, uploadedFilePath, c.ClientIP())
	// go routine'le calistirdigimiz icin addFuzz fonksiyonu db'ye ekleme yapmadan kullanıcıya fuzzları donuyor. guncel eklenen fuzz donmuyor
	// bu yuzden 1 sn bekleyelim :) #TODO daha iyi bi cozum
	time.Sleep(1 * time.Second)
	fuzzs, err := models.GetFuzzs()
	if err != nil {
		fmt.Println(err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "I think somethings have just broken. :(",
		})
		return
	}

	fuzzTemps, err := returnFuzzTemps(fuzzs)
	if err != nil {
		fmt.Println(err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "I think somethings have just broken. :(",
		})
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "go-fuzzer",
		"Fuzzs": fuzzTemps,
	})
}

func addFuzz(url string, wordlistFile string, ip string) {
	// burada insert et fuzz'ı
	fuzz := models.Fuzz{
		Url:          url,
		WordlistFile: wordlistFile,
		OutputFile:   "",
		Ip:           ip,
		WordCount:    -1,
		Started:      makeTimestamp(time.Now()),
		Ended:        -1,
		Finished:     0,
		Error:        0,
	}
	len, err := wc(wordlistFile)
	if err != nil {
		fuzz.Error = 1
		models.UpdateFuzz(fuzz)
		fmt.Println("go addFuzz: ", err)
		return
	}
	fuzz.WordCount = len

	fuzz, err = models.InsertFuzz(fuzz)
	if err != nil {
		fmt.Println("go addFuzz: ", err)
		return
	}

	out, err := exec.Command("ffuf", "-u", url, "-w", wordlistFile, "-p", "1-3").Output()
	if err != nil {
		fuzz.Error = 1
		models.UpdateFuzz(fuzz)
		fmt.Println("go addFuzz: ", err)
		return
	}

	outputFile := DIR_FFUFOUTPUTS + strconv.Itoa(fuzz.Id)
	err = os.WriteFile(outputFile, out, 0644)
	if err != nil {
		fuzz.Error = 1
		models.UpdateFuzz(fuzz)
		fmt.Println("go addFuzz: ", err)
		return
	}

	fuzz.Ended = makeTimestamp(time.Now())
	fuzz.Finished = 1
	fuzz.OutputFile = outputFile

	fuzz, err = models.UpdateFuzz(fuzz)
	if err != nil {
		fmt.Println("go addFuzz: ", err)
		return
	}

	//TODO bu boyle olmaz event-handler sekli olmalı

	fmt.Println(string(out))
}

func output(c *gin.Context) {
	if FLAG_HOSTCNTRL && c.Request.Header[KEY_AUTH_HEADER] == nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusNotFound,
			"error_message": "Not Found. :)",
		})
		return
	}
	id := c.Param("id")
	id_int, err := strconv.Atoi(id)
	fuzz, err := models.GetFuzz(id_int)
	if err != nil {
		fmt.Println("output: ", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"title":         "go-fuzzer",
			"status_code":   http.StatusInternalServerError,
			"error_message": "I think somethings have just broken. :(",
		})
		return
	}
	dat, err := os.ReadFile(fuzz.OutputFile)
	fmt.Println(fuzz.OutputFile)
	if err != nil {
		fmt.Println("output :", err)
	}
	c.HTML(http.StatusOK, "output.html", gin.H{
		"url":      fuzz.Url,
		"wordlist": fuzz.WordlistFile,
		"data":     string(dat)})

}

func makeTimestamp(time time.Time) int64 {
	return time.UnixNano() / 1e6
}

func wc(fileName string) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Err ", err)
		return -1, err
	}
	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	return lines, nil
}
