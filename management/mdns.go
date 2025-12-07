package management

import (
	"context"
	"log/slog"
	"sync"

	"github.com/grandcat/zeroconf"
)

// MdnsService manages the mDNS broadcast.
type MdnsService struct {
	sync.Mutex
	server   *zeroconf.Server
	Enabled  bool
	Name     string
	Port     int
	log      *slog.Logger
	shutdown context.CancelFunc
}

// NewMdnsService creates a new mDNS service manager.
func NewMdnsService(log *slog.Logger) *MdnsService {
	return &MdnsService{
		log: log,
	}
}

// Configure updates the configuration and restarts the service if enabled.
func (s *MdnsService) Configure(enabled bool, name string, port int) error {
	s.Lock()
	defer s.Unlock()

	// If no changes, do nothing (optional optimization)
	if s.Enabled == enabled && s.Name == name && s.Port == port {
		return nil
	}

	// Stop existing if running
	if s.server != nil {
		s.stop()
	}

	s.Enabled = enabled
	s.Name = name
	s.Port = port

	if s.Enabled {
		return s.start()
	}
	return nil
}

// start registers the mDNS service.
// Caller must hold lock.
func (s *MdnsService) start() error {
	if s.Name == "" {
		s.Name = "Mochi MQTT"
	}
	if s.Port == 0 {
		s.Port = 1883 // Default MQTT port
	}

	s.log.Info("starting mdns service", "name", s.Name, "port", s.Port)

	// _mqtt._tcp
	server, err := zeroconf.Register(s.Name, "_mqtt._tcp", "local.", s.Port, []string{"txtv=0", "lo=1", "la=2"}, nil)
	if err != nil {
		s.log.Error("failed to start mdns service", "error", err)
		return err
	}

	s.server = server
	return nil
}

// stop shuts down the mDNS service.
// Caller must hold lock.
func (s *MdnsService) stop() {
	if s.server != nil {
		s.log.Info("stopping mdns service")
		s.server.Shutdown()
		s.server = nil
	}
}

// Stop safely stops the service (public method).
func (s *MdnsService) Stop() {
	s.Lock()
	defer s.Unlock()
	s.stop()
}

// Config returns the current configuration.
func (s *MdnsService) Config() (bool, string, int) {
	s.Lock()
	defer s.Unlock()
	return s.Enabled, s.Name, s.Port
}
