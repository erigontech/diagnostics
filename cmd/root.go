package cmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/ledgerwatch/diagnostics/assets"
)

var (
	// Used for flags.
	cfgFile    string
	listenAddr string
	listenPort int

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

var validPath = regexp.MustCompile("^/([a-zA-Z0-9]+)$")

func (dh *DiagHandler) giveRequestsHandler(w http.ResponseWriter, r *http.Request) {
loop:
	for {
		select {
		case request := <-dh.requestCh:
			fmt.Fprintf(w, "%s\n", request)
		default:
			break loop
		}
	}
}

func takeResponsesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Take responses %s", r.URL.Path)
}

func showHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Show %s", r.URL.Path)
}

type DiagHandler struct {
	// Requests can be of arbitrary types, they get converted to a string and sent to the support subcommand
	requestCh chan interface{}
	cancel    context.CancelFunc
}

// Conn is client/server symmetric connection.
// It implements the io.Reader/io.Writer/io.Closer to read/write or close the connection to the other side.
// It also has a Send/Recv function to use channels to communicate with the other side.
type Conn struct {
	r  io.Reader
	wc io.WriteCloser

	cancel context.CancelFunc

	wLock sync.Mutex
	rLock sync.Mutex
}

func newConn(ctx context.Context, r io.Reader, wc io.WriteCloser) (*Conn, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &Conn{
		r:      r,
		wc:     wc,
		cancel: cancel,
	}, ctx
}

// Write writes data to the connection
func (c *Conn) Write(data []byte) (int, error) {
	c.wLock.Lock()
	defer c.wLock.Unlock()
	return c.wc.Write(data)
}

// Read reads data from the connection
func (c *Conn) Read(data []byte) (int, error) {
	c.rLock.Lock()
	defer c.rLock.Unlock()
	return c.r.Read(data)
}

// Close closes the connection
func (c *Conn) Close() error {
	c.cancel()
	return c.wc.Close()
}

type flushWrite struct {
	w io.Writer
	f http.Flusher
}

func (w *flushWrite) Write(data []byte) (int, error) {
	n, err := w.w.Write(data)
	w.f.Flush()
	return n, err
}

func (w *flushWrite) Close() error {
	// Currently server side close of connection is not supported in Go.
	// The server closes the connection when the http.Handler function returns.
	// We use connection context and cancel function as a work-around.
	return nil
}

var ErrHTTP2NotSupported = "HTTP2 not supported"

func (dh *DiagHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := validPath.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	if !r.ProtoAtLeast(2, 0) {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, ErrHTTP2NotSupported, http.StatusHTTPVersionNotSupported)
		return
	}
	c, ctx := newConn(r.Context(), r.Body, &flushWrite{w: w, f: flusher})
	defer c.Close()

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var writeBuf bytes.Buffer
	for {
		fmt.Fprintf(&writeBuf, "/db/list\n")
		if _, err := c.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v", err)
			return
		}
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
		fmt.Printf("RESPONSE: \n%s\n", metricsBuf)
	}
}

func webServer() error {
	sigs := make(chan os.Signal, 1)
	interruptCh := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigs
		cancel()
	}()
	h2s := &http2.Server{
		// ...
	}
	mux := http.NewServeMux()
	mux.Handle("/ui/", http.FileServer(http.FS(assets.Content)))
	dh := &DiagHandler{requestCh: make(chan interface{}, 1024)}
	mux.Handle("/", dh)
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", listenAddr, listenPort),
		Handler:        h2c.NewHandler(mux, h2s),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		ConnContext: func(_ context.Context, _ net.Conn) context.Context {
			return ctx
		},
	}
	interrupt := false
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Printf("Running server problem: %v", err)
		}
	}()
	pollInterval := 500 * time.Millisecond
	pollEvery := time.NewTicker(pollInterval)
	defer pollEvery.Stop()
	for !interrupt {
		select {
		case interrupt = <-interruptCh:
			log.Printf("interrupted, please wait for cleanup")
		case <-pollEvery.C:
		}
	}
	s.Shutdown(context.Background())
	return nil
}
