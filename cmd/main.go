// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2022 mochi-mqtt, mochi-co
// SPDX-FileContributor: mochi-co

package main

import (
	"crypto/tls"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/hooks/storage/bolt"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/mochi-mqtt/server/v2/management"
)

func main() {
	// 1. Check/Create .env
	if _, err := os.Stat(".env"); os.IsNotExist(err) {
		defaultEnv := `MQTT_PORT=:1883
WS_PORT=:1882
MGMT_PORT=:8888
INFO_PORT=:8080
`
		if err := os.WriteFile(".env", []byte(defaultEnv), 0644); err != nil {
			log.Fatal("failed to create default .env file: ", err)
		}
		log.Println("Created default .env file")
	}

	// 2. Load .env
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	tcpAddr := os.Getenv("MQTT_PORT")
	wsAddr := os.Getenv("WS_PORT")
	mgmtAddr := os.Getenv("MGMT_PORT")

	// Optional: TLS support can be added via env vars too if needed, keeping simple for now or using flags as secondary?
	// User said "env config... through env file config mqtt port, ws port, mgmt port, info port"
	// Keeping it simple as requested. TLS flags can remain or be moved to env if strictly required, but usually certificates paths result in long env vars.
	// Let's keep existing flags for TLS as optional overrides or just remove them to strictly follow "parameters too many".
	// The user complained "parameters too many", so minimizing flags is good.
	// But let's leave TLS flags for now as they weren't explicitly banned, just "wanted env config for ports".

	tlsCertFile := flag.String("tls-cert-file", "", "TLS certificate file")
	tlsKeyFile := flag.String("tls-key-file", "", "TLS key file")
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	var tlsConfig *tls.Config

	if tlsCertFile != nil && tlsKeyFile != nil && *tlsCertFile != "" && *tlsKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(*tlsCertFile, *tlsKeyFile)
		if err != nil {
			return
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	server := mqtt.New(nil)
	authHook := new(auth.Hook)
	_ = server.AddHook(authHook, nil)

	// Storage Hook (BoltDB)
	storageHook := new(bolt.Hook)
	err = server.AddHook(storageHook, &bolt.Options{
		Path:    "data.db", // "Default to same folder as binary"
		Options: nil,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Wire storage to auth hook for persistence
	authHook.SetStorage(storageHook)

	// Settings
	settings := management.NewSettingsManager()

	// Sync Env to Settings if first run/override?
	// For simplicity, let's respect Settings file if exists.
	// If mdns enabled in env, ensure it's enabled in settings?
	if os.Getenv("MDNS_ENABLED") == "true" {
		mdnsCfg := settings.GetMDNS()
		// Only update if not already enabled? Or force?
		// Let's force update from Env to ensure Docker env vars work.
		mdnsCfg.Enabled = true
		if name := os.Getenv("MDNS_NAME"); name != "" {
			mdnsCfg.Name = name
		}
		_ = settings.UpdateMDNS(mdnsCfg)
	}

	// mDNS Service
	mdns := management.NewMdnsService(server.Log)
	mdnsCfg := settings.GetMDNS()
	_ = mdns.Configure(mdnsCfg.Enabled, mdnsCfg.Name, mdnsCfg.Port)

	// TLS Listener from Settings
	tlsSettings := settings.GetTLS()
	if tlsSettings.Enabled && tlsSettings.Cert != "" && tlsSettings.Key != "" {
		cert, err := tls.X509KeyPair([]byte(tlsSettings.Cert), []byte(tlsSettings.Key))
		if err == nil {
			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
			tcp := listeners.NewTCP(listeners.Config{
				ID:        "mqtts",
				Address:   tlsSettings.Port,
				TLSConfig: tlsConfig,
			})
			_ = server.AddListener(tcp)
		} else {
			server.Log.Error("failed to load stored tls config", "error", err)
		}
	}

	if mgmtAddr != "" {
		mgmt := management.New(listeners.Config{
			ID:      "mgmt",
			Address: mgmtAddr,
		}, server, authHook, storageHook, mdns, settings)
		err := server.AddListener(mgmt)
		if err != nil {
			log.Fatal(err)
		}
	}

	if tcpAddr != "" {
		tcp := listeners.NewTCP(listeners.Config{
			ID:        "t1",
			Address:   tcpAddr,
			TLSConfig: tlsConfig,
		})
		err := server.AddListener(tcp)
		if err != nil {
			log.Fatal(err)
		}
	}

	if wsAddr != "" {
		ws := listeners.NewWebsocket(listeners.Config{
			ID:      "ws1",
			Address: wsAddr,
		})
		err := server.AddListener(ws)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Removed separate info listener as requested ("info functionality merged in mgmt")

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-done
	server.Log.Warn("caught signal, stopping...")
	_ = server.Close()
	server.Log.Info("mochi mqtt shutdown complete")
}
