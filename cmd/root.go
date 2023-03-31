package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"

	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ledgerwatch/diagnostics/assets"
)

var (
	// Used for flags.
	cfgFile        string
	listenAddr     string
	listenPort     int
	serverKeyFile  string
	serverCertFile string
	caCertFiles    []string

	rootCmd = &cobra.Command{
		Use:   "diagnostics",
		Short: "Diagnostics web server for Erigon support",
		Long:  `Diagnostics web server for Erigon support`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return webServer()
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.Flags().StringVar(&listenAddr, "addr", "localhost", "network interface to listen on")
	rootCmd.Flags().IntVar(&listenPort, "port", 8080, "port to listen on")
	rootCmd.Flags().StringVar(&serverKeyFile, "tls.key", "", "path to server TLS key")
	rootCmd.MarkFlagRequired("tls.key")
	rootCmd.Flags().StringVar(&serverCertFile, "tls.cert", "", "paths to server TLS certificates")
	rootCmd.MarkFlagRequired("tls.cert")
	rootCmd.Flags().StringSliceVar(&caCertFiles, "tls.cacerts", []string{}, "comma-separated list of paths to and CAs TLS certificates")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cobra")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

type BridgeHandler struct {
	cancel context.CancelFunc
	sh     *SessionHandler
}

var supportUrlRegex = regexp.MustCompile("^/support/([0-9]+)$")

var ErrHTTP2NotSupported = "HTTP2 not supported"

func (bh *BridgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !r.ProtoAtLeast(2, 0) {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	m := supportUrlRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	pin, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil {
		http.Error(w, "Error parsing session PIN", http.StatusNotFound)
		log.Printf("Errir parsing session pin %s: %v", m[1], err)
		return
	}
	fmt.Printf("PIN: %d\n", pin)
	_, ok = bh.sh.findSession(pin)
	if !ok {
		http.Error(w, fmt.Sprintf("Session with specified PIN %d not found", pin), http.StatusNotFound)
		log.Printf("Session with specified PIN %d not found", pin)
		return
	}
	fmt.Fprintf(w, "SUCCESS\n")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var writeBuf bytes.Buffer
	for {
		//fmt.Printf("Sending request\n")
		writeBuf.Reset()
		fmt.Fprintf(&writeBuf, "/db/list\n")
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v", err)
			return
		}
		flusher.Flush()
		// Read the response
		var sizeBuf [4]byte
		if _, err := io.ReadFull(r.Body, sizeBuf[:]); err != nil {
			log.Printf("Reading size of metrics response: %v", err)
			return
		}
		metricsBuf := make([]byte, binary.BigEndian.Uint32(sizeBuf[:]))
		if _, err := io.ReadFull(r.Body, metricsBuf); err != nil {
			log.Printf("Reading metrics response: %v", err)
			return
		}
		//fmt.Printf("RESPONSE: \n%s\n", metricsBuf)
	}
}

type Session struct {
}

type SessionHandler struct {
	sessionsLock sync.Mutex
	sessions     map[uint64]*Session
}

func (sh *SessionHandler) allocateNewSession() (uint64, *Session) {
	sh.sessionsLock.Lock()
	defer sh.sessionsLock.Unlock()
	pin := uint64(rand.Int63n(100_000_000))
	for _, ok := sh.sessions[pin]; ok; _, ok = sh.sessions[pin] {
		pin = uint64(rand.Int63n(100_000_000))
	}
	session := &Session{}
	sh.sessions[pin] = session
	return pin, session
}

func (sh *SessionHandler) findSession(pin uint64) (*Session, bool) {
	sh.sessionsLock.Lock()
	defer sh.sessionsLock.Unlock()
	session, ok := sh.sessions[pin]
	return session, ok
}

var sessionUrlRegex = regexp.MustCompile("^/session/(new_session|resume_session)$")

func (sh *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := sessionUrlRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Error parsing form: %v", err)
		return
	}
	switch m[1] {
	case "new_session":
		// Generate new session PIN that does not exist yet
		pin, session := sh.allocateNewSession()
		fmt.Fprintf(w, "NEW_SESSION %d %p\n", pin, session)
	case "resume_session":
		pinStr := r.FormValue("pin")
		sessionPin, err := strconv.ParseUint(pinStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Error parsing session PIN %s: %v", pinStr, err)
			return
		}
		session, ok := sh.findSession(sessionPin)
		if !ok {
			fmt.Fprintf(w, "Session %d is not found", sessionPin)
			return
		}
		fmt.Fprintf(w, "RESUME SESSION: %d %p\n", sessionPin, session)
	}
}

func webServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.FileServer(http.FS(assets.Content)))
	sh := &SessionHandler{
		sessions: map[uint64]*Session{},
	}
	mux.Handle("/session/", sh)
	bh := &BridgeHandler{sh: sh}
	mux.Handle("/support/", bh)
	certPool := x509.NewCertPool()
	for _, caCertFile := range caCertFiles {
		caCert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			log.Printf("Reading server certificate: %v", err)
			return err
		}
		certPool.AppendCertsFromPEM(caCert)
	}
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", listenAddr, listenPort),
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		ConnContext: func(_ context.Context, _ net.Conn) context.Context {
			return ctx
		},
		TLSConfig: tlsConfig,
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		s.Shutdown(ctx)
	}()
	err := s.ListenAndServeTLS(serverCertFile, serverKeyFile)
	if err != nil {
		log.Printf("Running server problem: %v", err)
	}
	return nil
}
