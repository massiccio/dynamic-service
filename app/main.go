package main

import (
  "github.com/gin-gonic/gin"
  "log"
  "os"
  "net/http"
  "net"
  "time"
  "math/rand"
  "fmt"
  "bytes"
  "encoding/json"
  "strconv"
)

type ConsulService struct {
  ID      string            `json:"ID"`
  Name    string            `json:"Name"`
  Tags    []string          `json:"Tags"`
  Address string            `json:"Address"`
  Port    int               `json:"Port"`
  Check   map[string]string `json:"Check"`
}

func registerService() {
  consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
  serviceName := os.Getenv("SERVICE_NAME")
  servicePort := os.Getenv("SERVICE_PORT")

  hostname, _ := os.Hostname()  // Get container ID for unique registration

  serviceID := fmt.Sprintf("%s-%s", serviceName, hostname)
  serviceIP := os.Getenv("SERVICE_IP")  // Optionally pass if known, or discover dynamically

  if serviceIP == "" {
      serviceIP = getOutboundIP()  // Get container IP dynamically
  }

  payload := ConsulService{
      ID:      serviceID,
      Name:    serviceName,
      Tags:    []string{"go-service"},
      Address: serviceIP,
      Port:    atoi(servicePort),
      Check: map[string]string{
          "http": fmt.Sprintf("http://%s:%s/ping", serviceIP, servicePort),
          "interval": "30s",
          "timeout": "5s",
          "deregistercriticalserviceafter": "1m",
      },
  }

  data, _ := json.Marshal(payload)

  req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/agent/service/register", consulAddr), bytes.NewBuffer(data))
  req.Header.Set("Content-Type", "application/json")
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
      fmt.Printf("Failed to register service: %s\n", err)
      return
  }
  defer resp.Body.Close()

  fmt.Println("Service registered with Consul")
}

func getOutboundIP() string {
  conn, _ := net.Dial("udp", "8.8.8.8:80")
  defer conn.Close()
  localAddr := conn.LocalAddr().(*net.UDPAddr)
  return localAddr.IP.String()
}

func atoi(str string) int {
  i, _ := strconv.Atoi(str)
  return i
}

func main() {
    registerService()
    rand.Seed(time.Now().UnixNano())

    gin.SetMode(gin.ReleaseMode)
    gin.DefaultWriter = os.Stdout  // Optional: If you want to log somewhere else

    // Create a new Gin router
    // r := gin.New()
    r := gin.Default()

    // r.GET("/health", func(c *gin.Context) {
    //   c.JSON(http.StatusOK, gin.H{"status": "healthy"})
    // })

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
