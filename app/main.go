package main

import (
  "github.com/gin-gonic/gin"
  consulapi "github.com/hashicorp/consul/api"
  "log"
  "os"
  "os/signal"
  "syscall"
  "net/http"
  "time"
  "math/rand"
  "fmt"
  "net"
  "strconv"
)

func deregisterServiceWithConsul(serviceID string) {
  config := consulapi.DefaultConfig()
  config.Address = os.Getenv("CONSUL_HTTP_ADDR")
  client, err := consulapi.NewClient(config)
  if err != nil {
      log.Fatalf("Error creating Consul client: %v", err)
  }

  err = client.Agent().ServiceDeregister(serviceID)
  if err != nil {
      log.Fatalf("Error deregistering service from Consul: %v", err)
  }
  log.Printf("Service %s deregistered from Consul", serviceID)
}

func getServiceID() string {
  serviceName := os.Getenv("SERVICE_NAME")
  hostname, _ := os.Hostname()  // Get container ID for unique registration 
  serviceID := fmt.Sprintf("%s-%s", serviceName, hostname)// Unique ID
  return serviceID
}

func registerServiceWithConsul(s *service) {
  consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
  serviceTags := []string{"golang-service"} // You can add more tags for the service
  serviceIP := os.Getenv("SERVICE_IP")  // Optionally pass if known, or discover dynamically

  if serviceIP == "" {
      serviceIP = getOutboundIP()  // Get container IP dynamically
  }

  // Create a new Consul client
  config := consulapi.DefaultConfig()
  config.Address = consulAddr
  client, err := consulapi.NewClient(config)
  if err != nil {
      log.Fatalf("Error creating Consul client: %v", err)
  }

  // Define the health check for the service
  healthCheck := &consulapi.AgentServiceCheck{
      HTTP:                           fmt.Sprintf("http://%s:%s/ping", serviceIP, s.port),
      Interval:                       "10s",
      Timeout:                        "5s",
      DeregisterCriticalServiceAfter: "1m",
  }

  // Register the service with Consul
  serviceRegistration := &consulapi.AgentServiceRegistration{
      ID:      s.id,
      Name:    s.name,
      Tags:    serviceTags,
      Address: serviceIP, // The address of your Golang service
      Port:    atoi(s.port), // The port your service listens on
      Check:   healthCheck, // Health check for the service
  }

  err = client.Agent().ServiceRegister(serviceRegistration)
  if err != nil {
      log.Fatalf("Error registering service with Consul: %v", err)
  }
  log.Printf("Service %s registered with Consul", s.name)
}

func getOutboundIP() string {
  conn, _ := net.Dial("udp", "8.8.8.8:80")
  defer conn.Close()
  localAddr := conn.LocalAddr().(*net.UDPAddr)
  return localAddr.IP.String()
}

func atoi(str string) int {
  i, err := strconv.Atoi(str)
  if err != nil {
    return 18080
  }
  return i
}

type service struct {
  name string
  id string
  port string
}

func getService() *service {
  serviceName := os.Getenv("SERVICE_NAME")
  serviceID := getServiceID()
  servicePort := os.Getenv("SERVICE_PORT")

  s := service{
    name: serviceName,
    id: serviceID,
    port: servicePort,
  }
  return &s
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

    rand.Seed(time.Now().UnixNano())

    gin.SetMode(gin.ReleaseMode)
    gin.DefaultWriter = os.Stdout  // Optional: If you want to log somewhere else

    // Create a new Gin router
    // r := gin.New()
    r := gin.Default()

    r.GET("/ping", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
          "message": "Pong!",
          "serviceID": s.id,
        })
      })

    r.GET("/pong", func(c *gin.Context) {
        switch rand.Intn(2) {
        case 0:
            c.JSON(http.StatusOK, gin.H{
            "message": "Ping!",
            "serviceID": s.id,
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
