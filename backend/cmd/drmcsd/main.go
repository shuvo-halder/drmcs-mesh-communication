package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/drmcs/backend/internal/alerts"
	"github.com/drmcs/backend/internal/crypto"
	"github.com/drmcs/backend/internal/discovery"
	"github.com/drmcs/backend/internal/fileshare"
	"github.com/drmcs/backend/internal/messaging"
	"github.com/drmcs/backend/internal/routing"
	"github.com/drmcs/backend/internal/storage"
)

func main() {
	port := flag.Int("port", 8080, "Listening port")
	nodeName := flag.String("name", "", "Node name (default: hostname)")
	dataDir := flag.String("data", "./data", "Data directory for storage")
	flag.Parse()

	if *nodeName == "" {
		hostname, _ := os.Hostname()
		*nodeName = hostname
	}

	// Initialize encryption
	cryptoKey, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	// Initialize SQLite storage
	dbPath := fmt.Sprintf("%s/drmcs.db", *dataDir)
	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize routing module
	router := routing.NewAODVRouter(*nodeName)

	// Initialize peer discovery
	discoveryPort := *port + 1
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{Port: discoveryPort})
	if err != nil {
		log.Fatalf("Failed to listen on discovery port: %v", err)
	}
	defer listener.Close()

	discoveryService := discovery.NewService(*nodeName, discoveryPort, cryptoKey.PublicKey)
	go discoveryService.Start(listener)

	// Initialize messaging
	msgHandler := messaging.NewHandler(*nodeName, router, cryptoKey.PrivateKey, store)
	go msgHandler.Start(*port)

	// Initialize alerts
	alertSystem := alerts.NewSystem(*nodeName, msgHandler, store)
	go alertSystem.Start()

	// Initialize file sharing
	fileTransfer := fileshare.NewTransfer(*nodeName, store, cryptoKey.PrivateKey)
	go fileTransfer.Start(*port + 2)

	log.Printf("DRMCS Node '%s' started on port %d (discovery: %d, files: %d)",
		*nodeName, *port, discoveryPort, *port+2)
	log.Printf("Public key: %x...", cryptoKey.PublicKey[:8])

	// Print node info
	addrs := getLocalAddresses()
	for _, addr := range addrs {
		log.Printf("Node reachable at: %s:%d", addr, *port)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down DRMCS node...")
	discoveryService.Stop()
	msgHandler.Stop()
	alertSystem.Stop()
	fileTransfer.Stop()
	log.Println("Shutdown complete.")
}

func getLocalAddresses() []string {
	var addrs []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return addrs
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrsList, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrsList {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				addrs = append(addrs, ip.String())
			}
		}
	}
	return addrs
}