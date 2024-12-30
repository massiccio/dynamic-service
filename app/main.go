package main

import (
  "github.com/gin-gonic/gin"
  "github.com/hashicorp/consul/api"
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
  config := api.DefaultConfig()
  config.Address = os.Getenv("CONSUL_HTTP_ADDR")
  client, err := api.NewClient(config)
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

func registerServiceWithConsul(serviceID string) {
  consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
  serviceName := os.Getenv("SERVICE_NAME")
  servicePort := os.Getenv("SERVICE_PORT")

  // serviceID := getServiceId()
  serviceTags := []string{"golang-service"} // You can add more tags for the service
  serviceIP := os.Getenv("SERVICE_IP")  // Optionally pass if known, or discover dynamically

  if serviceIP == "" {
      serviceIP = getOutboundIP()  // Get container IP dynamically
  }

  // Create a new Consul client
  config := api.DefaultConfig()
  config.Address = consulAddr
  client, err := api.NewClient(config)
  if err != nil {
      log.Fatalf("Error creating Consul client: %v", err)
  }

  // Define the health check for the service
  healthCheck := &api.AgentServiceCheck{
      HTTP:                           fmt.Sprintf("http://%s:%s/ping", serviceIP, servicePort),
      Interval:                       "10s",
      Timeout:                        "5s",
      DeregisterCriticalServiceAfter: "1m",
  }

  // Register the service with Consul
  serviceRegistration := &api.AgentServiceRegistration{
      ID:      serviceID,
      Name:    serviceName,
      Tags:    serviceTags,
      Address: serviceIP, // The address of your Golang service
      Port:    atoi(servicePort), // The port your service listens on
      Check:   healthCheck, // Health check for the service
  }

  err = client.Agent().ServiceRegister(serviceRegistration)
  if err != nil {
      log.Fatalf("Error registering service with Consul: %v", err)
  }
  log.Printf("Service %s registered with Consul", serviceName)
}

// func registerService() {
//   consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
//   serviceName := os.Getenv("SERVICE_NAME")
//   servicePort := os.Getenv("SERVICE_PORT")

//   hostname, _ := os.Hostname()  // Get container ID for unique registration

//   serviceID := fmt.Sprintf("%s-%s", serviceName, hostname)
//   serviceIP := os.Getenv("SERVICE_IP")  // Optionally pass if known, or discover dynamically

//   if serviceIP == "" {
//       serviceIP = getOutboundIP()  // Get container IP dynamically
//   }

//   payload := ConsulService{
//       ID:      serviceID,
//       Name:    serviceName,
//       Tags:    []string{"go-service"},
//       Address: serviceIP,
//       Port:    atoi(servicePort),
//       Check: map[string]string{
//           "http": fmt.Sprintf("http://%s:%s/ping", serviceIP, servicePort),
//           "interval": "30s",
//           "timeout": "5s",
//           "deregistercriticalserviceafter": "1m",
//       },
//   }

//   data, _ := json.Marshal(payload)

//   req, err := http.NewRequest("PUT", fmt.Sprintf("%s/v1/agent/service/register", consulAddr), bytes.NewBuffer(data))
//   req.Header.Set("Content-Type", "application/json")
//   resp, err := http.DefaultClient.Do(req)
//   if err != nil {
//       fmt.Printf("Failed to register service: %s\n", err)
//       return
//   }
//   defer resp.Body.Close()

//   fmt.Println("Service registered with Consul")
// }

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
    serviceID := getServiceID()
    registerServiceWithConsul(serviceID)

    // Handle graceful shutdown
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigs
        deregisterServiceWithConsul(serviceID)
    }()

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
