package main

import (
    "github.com/gin-gonic/gin"
    "log"
    "os"
    "net/http"
    "time"
    "math/rand"
)

func main() {
    rand.Seed(time.Now().UnixNano())

    gin.SetMode(gin.ReleaseMode)
    gin.DefaultWriter = os.Stdout  // Optional: If you want to log somewhere else

    // Create a new Gin router
    // r := gin.New()
    r := gin.Default()

    // Define the /ping route
    // r.GET("/ping", func(c *gin.Context) {
    //     c.String(200, "Pong!")
    // })
    r.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
          "message": "Pong!",
        })
      })

    // Define the /pong route
    // r.GET("/pong", func(c *gin.Context) {
    //     c.String(200, "Ping!")
    // })
    r.GET("/pong", func(c *gin.Context) {
        switch rand.Intn(2) {
		case 0:
            c.JSON(http.StatusOK, gin.H{
            "message": "Ping!",
            })
        case 1:
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": "Recovered Internal Server Error",
            })
        }
      })

    // Start the server
    log.Println("Server is running on port 18080...")
    r.Run("0.0.0.0:18080")
}
