package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

// Server interface defines the required methods for a server
type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, req *http.Request)
}

// simpleServer struct represents a single backend server
type simpleServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

// newSimpleServer creates a new instance of simpleServer
func newSimpleServer(addr string) *simpleServer {
	serverURL, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}

	return &simpleServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

func (s *simpleServer) Address() string {
	return s.addr
}

// IsAlive checks the server health by sending a GET request
func (s *simpleServer) IsAlive() bool {
	resp, err := http.Get(s.addr)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

// Serve forwards the request to the backend server
func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("Forwarding request to %s\n", s.addr)
	s.proxy.ServeHTTP(rw, req)
}

// LoadBalancer struct represents the load balancer
type LoadBalancer struct {
	port            string
	roundRobinIndex int
	serverList      []Server
	mu              sync.Mutex
}

// newLoadBalancer creates a new instance of LoadBalancer
func newLoadBalancer(port string, serverList []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinIndex: 0,
		serverList:      serverList,
	}
}

// getNextAvailableServer returns the next available server using round-robin algorithm
func (lb *LoadBalancer) getNextAvailableServer() Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	serverCount := len(lb.serverList)
	for i := 0; i < serverCount; i++ {
		server := lb.serverList[lb.roundRobinIndex%serverCount]
		lb.roundRobinIndex++
		if server.IsAlive() {
			fmt.Printf("Selected server: %s\n", server.Address())
			return server
		}
	}
	return nil
}

// serveProxy forwards the request to the selected backend server
func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	fmt.Printf("Received request: %s\n", req.URL.Path)
	targetServer := lb.getNextAvailableServer()
	if targetServer != nil {
		targetServer.Serve(rw, req)
	} else {
		http.Error(rw, "Service unavailable", http.StatusServiceUnavailable)
	}
}

func main() {
	serverList := []Server{
		newSimpleServer("https://www.instagram.com/"),
		newSimpleServer("https://www.twitter.com/"),
		newSimpleServer("https://www.medium.com/"),
	}

	lb := newLoadBalancer("8080", serverList)

	// Use ServeMux for better request handling
	mux := http.NewServeMux()
	mux.HandleFunc("/", lb.serveProxy)

	fmt.Printf("Load Balancer started at :%s\n", lb.port)
	err := http.ListenAndServe(":"+lb.port, mux)
	if err != nil {
		log.Fatal(err)
	}
}
