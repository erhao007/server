package management

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/hooks/storage/bolt"
	"github.com/mochi-mqtt/server/v2/listeners"
)

const (
	jwtSecret       = "mochi-mqtt-secret-key" // Should be in env, but hardcoded for demo or derived from env later
	accessTokenDur  = 15 * time.Minute
	refreshTokenDur = 24 * time.Hour * 7
)

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Management is a listener for the management interface.
type Management struct {
	sync.RWMutex
	id          string           // the internal id of the listener
	address     string           // the network address to bind to
	config      listeners.Config // configuration values for the listener
	listen      *http.Server     // the http server
	log         *slog.Logger     // server logger
	end         uint32           // ensure the close methods are only called once
	orgServer   *mqtt.Server     // reference to the main server instance
	authHook    *auth.Hook       // reference to the auth hook
	storageHook *bolt.Hook       // reference to the storage hook
	jwtKey      []byte           // key for signing JWTs
}

//go:embed dist/*
var distFS embed.FS

// New initializes and returns a new Management listener.
func New(config listeners.Config, server *mqtt.Server, authHook *auth.Hook, storageHook *bolt.Hook) *Management {
	// Simple secret retrieval, better from config
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = jwtSecret
	}

	return &Management{
		id:          config.ID,
		address:     config.Address,
		config:      config,
		orgServer:   server,
		authHook:    authHook,
		storageHook: storageHook,
		jwtKey:      []byte(secret),
	}
}

// ID returns the id of the listener.
func (l *Management) ID() string {
	return l.id
}

// Address returns the address of the listener.
func (l *Management) Address() string {
	return l.address
}

// Protocol returns the address of the listener.
func (l *Management) Protocol() string {
	if l.listen != nil && l.listen.TLSConfig != nil {
		return "https"
	}
	return "http"
}

// Init initializes the listener.
func (l *Management) Init(log *slog.Logger) error {
	l.log = log
	mux := http.NewServeMux()

	// Public Endpoints
	mux.HandleFunc("/api/v1/login", l.handleLogin)
	mux.HandleFunc("/api/v1/refresh", l.handleRefreshToken)
	mux.HandleFunc("/api/v1/install/check", l.handleInstallCheck)
	mux.HandleFunc("/api/v1/install", l.handleInstall)

	// Protected Endpoints
	mux.HandleFunc("/api/v1/listeners", l.authMiddleware(l.handleListeners))
	mux.HandleFunc("/api/v1/listeners/", l.authMiddleware(l.handleListenerDelete))
	mux.HandleFunc("/api/v1/users", l.authMiddleware(l.handleUsers))
	mux.HandleFunc("/api/v1/users/", l.authMiddleware(l.handleUserDelete))
	mux.HandleFunc("/api/v1/stats", l.authMiddleware(l.handleStats))

	// Storage Endpoints (Protected)
	mux.HandleFunc("/api/v1/storage/clients", l.authMiddleware(l.handleStoredClients))
	mux.HandleFunc("/api/v1/storage/clients/", l.authMiddleware(l.handleStoredClientDelete))
	mux.HandleFunc("/api/v1/storage/subscriptions", l.authMiddleware(l.handleStoredSubscriptions))
	mux.HandleFunc("/api/v1/storage/subscriptions/", l.authMiddleware(l.handleStoredSubscriptionDelete))
	mux.HandleFunc("/api/v1/storage/retained", l.authMiddleware(l.handleStoredRetained))
	mux.HandleFunc("/api/v1/storage/retained/", l.authMiddleware(l.handleStoredRetainedDelete))

	// Static UI serving (Embedded)
	// distFS serves the "dist" folder.
	// We need to strip the "dist" prefix to serve from root.
	fsys, err := fs.Sub(distFS, "dist")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(fsys))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If path starts with /api/, let the default handler (or 404) handle it.
		// However, standard mux matches longest pattern, so registered /api/ handlers will still trigger.
		// If we are here, it means no specific /api/ handler matched, OR it is a static file request.

		// If we want to serve index.html for all non-file non-api requests:
		path := r.URL.Path

		// check if file exists in fs
		f, err := fsys.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			// File exists, serve it
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File does not exist.
		// If it is an API call that failed to match, return 404.
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Otherwise, serve index.html (SPA Fallback)
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	l.listen = &http.Server{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Addr:         l.address,
		Handler:      mux,
	}

	if l.config.TLSConfig != nil {
		l.listen.TLSConfig = l.config.TLSConfig
	}

	return nil
}

const installLockFile = "install.lock"

func (l *Management) handleInstallCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := os.Stat(installLockFile)
	installed := !os.IsNotExist(err)
	if l.log != nil {
		l.log.Info("install check", "installed", installed, "error", err, "file", installLockFile)
	}

	l.jsonResponse(w, map[string]bool{"installed": installed}, http.StatusOK)
}

func (l *Management) handleInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Double check lock file
	if _, err := os.Stat(installLockFile); !os.IsNotExist(err) {
		l.jsonError(w, "installation already completed", http.StatusForbidden)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		l.jsonError(w, "username and password required", http.StatusBadRequest)
		return
	}

	ledger := l.authHook.Ledger()
	// Add Admin User
	if err := ledger.AddUser(req.Username, req.Password, true, "Super Admin", true); err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create Lock File
	f, err := os.Create(installLockFile)
	if err != nil {
		// Try to revert user creation? Or just fail.
		// If we fail here, the user exists but lock file doesn't.
		// Next try might match existing user or fail.
		// Let's just report error.
		l.jsonError(w, "failed to create lock file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	f.Close()

	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// Generate Tokens
func (l *Management) generateTokens(username string) (string, string, error) {
	// Access Token
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDur)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(l.jwtKey)
	if err != nil {
		return "", "", err
	}

	// Refresh Token
	refreshClaims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenDur)),
		},
	}
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(l.jwtKey)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenString, nil
}

// Serve starts listening for new connections and serving responses.
func (l *Management) Serve(establish listeners.EstablishFn) {
	var err error
	if l.listen.TLSConfig != nil {
		err = l.listen.ListenAndServeTLS("", "")
	} else {
		err = l.listen.ListenAndServe()
	}

	if err != nil && atomic.LoadUint32(&l.end) == 0 {
		l.log.Error("failed to serve management listener", "error", err, "listener", l.id)
	}
}

// Close closes the listener.
func (l *Management) Close(closeClients listeners.CloseFn) {
	l.Lock()
	defer l.Unlock()

	if atomic.CompareAndSwapUint32(&l.end, 0, 1) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = l.listen.Shutdown(ctx)
	}

	closeClients(l.id)
}

// ---------------- API Handlers ----------------

func (l *Management) jsonResponse(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func (l *Management) jsonError(w http.ResponseWriter, err string, status int) {
	l.jsonResponse(w, map[string]string{"error": err}, status)
}

// Listeners

func (l *Management) handleListeners(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ls := l.orgServer.Listeners.GetAll()
		resp := make([]map[string]any, 0, len(ls))
		for id, lst := range ls {
			resp = append(resp, map[string]any{
				"id":       id,
				"address":  lst.Address(),
				"protocol": lst.Protocol(),
				"type":     l.getListenerType(lst),
			})
		}
		l.jsonResponse(w, resp, http.StatusOK)

	case http.MethodPost:
		var req struct {
			Type    string `json:"type"`
			ID      string `json:"id"`
			Address string `json:"address"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			l.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		config := listeners.Config{
			Type:    req.Type,
			ID:      req.ID,
			Address: req.Address,
		}

		var lst listeners.Listener
		switch req.Type {
		case "tcp":
			lst = listeners.NewTCP(config)
		case "ws":
			lst = listeners.NewWebsocket(config)
		default:
			l.jsonError(w, "unsupported listener type", http.StatusBadRequest)
			return
		}

		if err := l.orgServer.AddListener(lst); err != nil {
			l.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Hack: Start serving the listener immediately!
		// server.AddListener puts it in the map, but we need to call Serve.
		// Use Listener.Serve(id, establisher)
		// How to get establisher?
		// We can get it from server.EstablishConnection methods.
		// server.EstablishConnection is func(listener string, c net.Conn) error.
		// listeners.EstablishFn is func(id string, c net.Conn) error.
		l.orgServer.Listeners.Serve(req.ID, l.orgServer.EstablishConnection)

		l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusCreated)

	default:
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (l *Management) handleListenerDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/listeners/")
	if id == "" {
		l.jsonError(w, "missing id", http.StatusBadRequest)
		return
	}

	// Stop listener
	l.orgServer.Listeners.Close(id, func(id string) {}) // CloseFn usually closes clients, we pass dummy or nil?
	// server.Listeners.Close takes (id, CloserFn).
	// We need to actually close the clients connected to this listener.
	// But `server` doesn't expose a method to close clients for a specific listener easily
	// except implicitly via the closer callback.
	// In server.go `Close` calls `closeClients`.
	// For now we might not be able to forcefully close existing clients if we don't have access to Server clients map.
	// But `Listeners.Close` calls `listener.Close(closer)`.
	// The listener implementation calls `closer(id)`.

	l.orgServer.Listeners.Delete(id)

	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (l *Management) getListenerType(val listeners.Listener) string {
	switch val.(type) {
	case *listeners.TCP:
		return "tcp"
	case *listeners.Websocket:
		return "ws"
	case *listeners.HTTPStats:
		return "stats"
	case *Management:
		return "management"
	default:
		return "unknown"
	}
}

// Users

func (l *Management) handleUsers(w http.ResponseWriter, r *http.Request) {
	ledger := l.authHook.Ledger()
	if ledger == nil {
		l.jsonError(w, "auth ledger not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		users := ledger.GetUsers()
		l.jsonResponse(w, users, http.StatusOK)

	case http.MethodPost, http.MethodPut:
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Allow    bool   `json:"allow"`
			Remarks  string `json:"remarks"`
			IsAdmin  bool   `json:"is_admin"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			l.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := ledger.AddUser(req.Username, req.Password, req.Allow, req.Remarks, req.IsAdmin); err != nil {
			l.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusCreated)

	default:
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (l *Management) handleUserDelete(w http.ResponseWriter, r *http.Request) {
	ledger := l.authHook.Ledger()
	if ledger == nil {
		l.jsonError(w, "auth ledger not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method != http.MethodDelete {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
	if err := ledger.RemoveUser(username); err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (l *Management) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Clone info to avoid race conditions if any (Server.Info is a pointer, but we should be careful.
	// system.Info usually has atomic counters or we just read it.
	// To be safe and consistent with http_sysinfo.go:
	info := *l.orgServer.Info.Clone()
	l.jsonResponse(w, info, http.StatusOK)
}

// Storage Handlers

func (l *Management) handleStoredClients(w http.ResponseWriter, r *http.Request) {
	if l.storageHook == nil {
		l.jsonError(w, "storage not initialized", http.StatusServiceUnavailable)
		return
	}

	clients, err := l.storageHook.StoredClients()
	if err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, clients, http.StatusOK)
}

func (l *Management) handleStoredClientDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/storage/clients/")
	if err := l.storageHook.DeleteClient(id); err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (l *Management) handleStoredSubscriptions(w http.ResponseWriter, r *http.Request) {
	if l.storageHook == nil {
		l.jsonError(w, "storage not initialized", http.StatusServiceUnavailable)
		return
	}

	subs, err := l.storageHook.StoredSubscriptions()
	if err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, subs, http.StatusOK)
}

func (l *Management) handleStoredSubscriptionDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Warning: parsing complex key ID from URL path might be tricky if it contains slashes.
	// But let's assume standard parsing or base64 encoding if needed.
	// We extract "clientID" and "filter" from query params or body might be safer?
	// But let's check how we named the key.
	// The id in DB is "SUB_client:filter".
	// But we need to pass clientID and filter to DeleteSubscription(clientID, filter).
	// Let's use Query params: ?client=...&filter=...
	clientID := r.URL.Query().Get("client")
	filter := r.URL.Query().Get("filter")
	if clientID == "" || filter == "" {
		l.jsonError(w, "missing client or filter", http.StatusBadRequest)
		return
	}

	if err := l.storageHook.DeleteSubscription(clientID, filter); err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (l *Management) handleStoredRetained(w http.ResponseWriter, r *http.Request) {
	if l.storageHook == nil {
		l.jsonError(w, "storage not initialized", http.StatusServiceUnavailable)
		return
	}

	msgs, err := l.storageHook.StoredRetainedMessages()
	if err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, msgs, http.StatusOK)
}

func (l *Management) handleStoredRetainedDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Topic can contain slashes. better use Query param ?topic=...
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		l.jsonError(w, "missing topic", http.StatusBadRequest)
		return
	}

	if err := l.storageHook.DeleteRetained(topic); err != nil {
		l.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	l.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// Middleware
func (l *Management) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 {
			http.Error(w, "invalid token format", http.StatusUnauthorized)
			return
		}

		tokenStr := bearerToken[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return l.jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (l *Management) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate against Ledger
	ledger := l.authHook.Ledger()
	users := ledger.GetUsers()

	authenticated := false
	for _, u := range users {
		if string(u.Username) == req.Username && string(u.Password) == req.Password {
			if !u.Disallow {
				// CHECK FOR ADMIN PRIVILEGE
				if !u.IsAdmin {
					l.jsonError(w, "insufficient privileges", http.StatusForbidden)
					return
				}
				authenticated = true
				break
			}
		}
	}

	if !authenticated {
		l.jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	access, refresh, err := l.generateTokens(req.Username)
	if err != nil {
		l.jsonError(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	l.jsonResponse(w, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	}, http.StatusOK)
}

func (l *Management) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		l.jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		l.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return l.jwtKey, nil
	})

	if err != nil || !token.Valid {
		l.jsonError(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}

	// Generate new tokens
	access, refresh, err := l.generateTokens(claims.Username)
	if err != nil {
		l.jsonError(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	l.jsonResponse(w, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	}, http.StatusOK)
}
