package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	consulapi "github.com/hashicorp/consul/api"
)

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

func registerServiceWithConsul(s *service) {
	consulAddr := os.Getenv("CONSUL_HTTP_ADDR")
	serviceTags := []string{"golang-service"} // You can add more tags for the service
	serviceIP := os.Getenv("SERVICE_IP")      // Optionally pass if known, or discover dynamically

	if serviceIP == "" {
		serviceIP = getOutboundIP() // Get container IP dynamically
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
		Address: serviceIP,    // The address of your Golang service
		Port:    atoi(s.port), // The port your service listens on
		Check:   healthCheck,  // Health check for the service
	}

	err = client.Agent().ServiceRegister(serviceRegistration)
	if err != nil {
		log.Fatalf("Error registering service with Consul: %v", err)
	}
	log.Printf("Service %s registered with Consul", s.name)
}
