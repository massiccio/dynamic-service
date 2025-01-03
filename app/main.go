package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func getServiceID() string {
	serviceName := os.Getenv("SERVICE_NAME")
	hostname, _ := os.Hostname()                             // Get container ID for unique registration
	serviceID := fmt.Sprintf("%s-%s", serviceName, hostname) // Unique ID
	return serviceID
}

type service struct {
	name      string
	id        string
	serviceIP string
	port      string
}

func getService() *service {
	serviceName := os.Getenv("SERVICE_NAME")
	serviceID := getServiceID()
	servicePort := os.Getenv("SERVICE_PORT")
	serviceIP := os.Getenv("SERVICE_IP") // Optionally pass if known, or discover dynamically

	if serviceIP == "" {
		serviceIP = getOutboundIP() // Get container IP dynamically
	}

	s := service{
		name:      serviceName,
		id:        serviceID,
		serviceIP: serviceIP,
		port:      servicePort,
	}
	return &s
}

func PingHandler(s *service) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Pong!",
			"serviceID": s.id,
			"serviceIP": s.serviceIP,
		})
	}
}

func PongHandler(s *service) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch val := rand.Intn(3); {
		case val < 2:
			c.JSON(http.StatusOK, gin.H{
				"message":   "Ping!",
				"serviceID": s.id,
				"serviceIP": s.serviceIP,
			})
		case val == 2:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Recovered Internal Server Error",
			})
		}
	}
}

func main() {
	s := getService()
	registerServiceWithConsul(s)

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		deregisterServiceWithConsul(s.id)
	}()

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = os.Stdout // Optional: If you want to log somewhere else

	// Create a new Gin router
	// r := gin.New()  # Disable logs
	r := gin.Default()

	r.GET("/ping", PingHandler(s))
	r.GET("/pong", PongHandler(s))

	// Start the server
	log.Println("Server is running on port 18080...")
	r.Run("0.0.0.0:18080")
}
