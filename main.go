package main

import (
	"crypto/md5"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"net/http"
	"strings"
)

type URLPair struct {
	gorm.Model
	Original string
	Short    string
}

const ApiVersion = "v1"
const BaseURL = "localhost"

func main() {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Println("Failed to connect to database: ", err)
		return
	}

	err = db.AutoMigrate(&URLPair{})
	if err != nil {
		log.Println("Failed to migrate database: ", err)
		return
	}

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	api := r.Group("/api/" + ApiVersion)
	{
		api.POST("/shorten", func(c *gin.Context) {
			type Data struct {
				URL string `json:"url"`
			}

			var data Data

			err := c.ShouldBindJSON(&data)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": err,
				})
				return
			}

			hash := md5.New()
			hash.Write([]byte(data.URL))
			shortId := base64.URLEncoding.EncodeToString(hash.Sum(nil))[:10]

			res := db.Create(&URLPair{
				Original: data.URL,
				Short:    shortId,
			})

			if res.Error != nil {
				log.Println(res.Error)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": err,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"shortened": BaseURL + "/" + shortId,
			})
		})
	}

	r.GET("/:id", func(c *gin.Context) {
		shortID := c.Param("id")

		var pair URLPair

		res := db.First(&pair, "short = ?", shortID)
		if res.Error != nil {
			log.Println(res.Error)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": err,
			})
			return
		}

		if !strings.HasPrefix(pair.Original, "https://") {
			pair.Original = "https://" + pair.Original
		}

		c.Redirect(http.StatusFound, pair.Original)
	})

	// TODO: Every 10 minutes, check for links that have been stored for over a week, and remove them
	go func() {

	}()

	r.StaticFile("/", "./static/index.html")

	err = r.Run(BaseURL + ":80")
	if err != nil {
		log.Println("Failed to start GIN router: ", err)
		return
	}

	log.Println("Exiting...")
}
