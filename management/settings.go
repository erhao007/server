package management

import (
	"encoding/json"
	"os"
	"sync"
)

const SettingsFile = "settings.json"

type TLSConfig struct {
	Enabled bool   `json:"enabled"`
	Port    string `json:"port"`
	Cert    string `json:"cert"` // PEM content
	Key     string `json:"key"`  // PEM content
}

type MDNSConfig struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Port    int    `json:"port"`
}

type AppSettings struct {
	MDNS MDNSConfig `json:"mdns"`
	TLS  TLSConfig  `json:"tls"`
}

type SettingsManager struct {
	sync.RWMutex
	Config AppSettings
}

func NewSettingsManager() *SettingsManager {
	sm := &SettingsManager{
		Config: AppSettings{
			MDNS: MDNSConfig{
				Enabled: false,
				Name:    "Mochi MQTT",
				Port:    1883,
			},
			TLS: TLSConfig{
				Enabled: false,
				Port:    ":8883",
			},
		},
	}
	// Try load
	_ = sm.Load()

	// If env vars exist and no settings file, override?
	// Or just use env as default if loaded file enabled is false?
	// Let's keep it simple: File > Defaults.
	// Env vars in main.go can populate this if we want initial setup.

	return sm
}

func (s *SettingsManager) Load() error {
	s.Lock()
	defer s.Unlock()

	data, err := os.ReadFile(SettingsFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.Config)
}

func (s *SettingsManager) Save() error {
	s.Lock()
	defer s.Unlock()

	data, err := json.MarshalIndent(s.Config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(SettingsFile, data, 0644)
}

func (s *SettingsManager) GetMDNS() MDNSConfig {
	s.RLock()
	defer s.RUnlock()
	return s.Config.MDNS
}

func (s *SettingsManager) UpdateMDNS(cfg MDNSConfig) error {
	s.Lock()
	s.Config.MDNS = cfg
	s.Unlock()
	return s.Save()
}

func (s *SettingsManager) GetTLS() TLSConfig {
	s.RLock()
	defer s.RUnlock()
	return s.Config.TLS
}

func (s *SettingsManager) UpdateTLS(cfg TLSConfig) error {
	s.Lock()
	s.Config.TLS = cfg
	s.Unlock()
	return s.Save()
}
