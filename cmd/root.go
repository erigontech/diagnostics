package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	weakrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"log"

	"github.com/google/btree"
	"github.com/ledgerwatch/diagnostics/assets"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	nodeSession, ok := bh.sh.findNodeSession(pin)
	if !ok {
		http.Error(w, fmt.Sprintf("Session with specified PIN %d not found", pin), http.StatusNotFound)
		log.Printf("Session with specified PIN %d not found", pin)
		return
	}
	fmt.Fprintf(w, "SUCCESS\n")
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer r.Body.Close()
	nodeSession.connect(r.RemoteAddr)
	defer nodeSession.disconnect()

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	var writeBuf bytes.Buffer
	for request := range nodeSession.requestCh {
		request.lock.Lock()
		url := request.url
		request.lock.Unlock()
		fmt.Printf("Sending request %s\n", url)
		writeBuf.Reset()
		fmt.Fprintf(&writeBuf, url)
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("writing metrics request: %v", err)
			request.retries++
			if request.retries < 16 {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		flusher.Flush()
		// Read the response
		var sizeBuf [4]byte
		if _, err := io.ReadFull(r.Body, sizeBuf[:]); err != nil {
			log.Printf("Reading size of metrics response: %v", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("reading size of metrics response: %v", err)
			request.retries++
			if request.retries < 16 {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		metricsBuf := make([]byte, binary.BigEndian.Uint32(sizeBuf[:]))
		if _, err := io.ReadFull(r.Body, metricsBuf); err != nil {
			log.Printf("Reading metrics response: %v", err)
			request.lock.Lock()
			request.served = true
			request.response = nil
			request.err = fmt.Sprintf("reading metrics response: %v", err)
			request.retries++
			if request.retries < 16 {
				select {
				case nodeSession.requestCh <- request:
				default:
				}
			}
			request.lock.Unlock()
			return
		}
		request.lock.Lock()
		request.served = true
		request.response = metricsBuf
		request.err = ""
		request.lock.Unlock()
	}
}

type NodeRequest struct {
	lock     sync.Mutex
	url      string
	served   bool
	response []byte
	err      string
	retries  int
}

const MaxRequestRetries = 16 // How many time to retry a request to the support

type NodeSession struct {
	lock       sync.Mutex
	sessionPin uint64
	Connected  bool
	RemoteAddr string
	requestCh  chan *NodeRequest // Channel for incoming metrics requests
}

func (ns *NodeSession) connect(remoteAddr string) {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = true
	ns.RemoteAddr = remoteAddr
}

func (ns *NodeSession) disconnect() {
	ns.lock.Lock()
	defer ns.lock.Unlock()
	ns.Connected = false
}

type UiNodeSession struct {
	SessionName string
	SessionPin  uint64
}

type UiSession struct {
	lock               sync.Mutex
	Session            bool
	SessionPin         uint64
	SessionName        string
	Errors             []string // Transient field - only filled for the time of template execution
	currentSessionName string
	NodeS              *NodeSession // Transient field - only filled for the time of template execution
	uiNodeTree         *btree.BTreeG[UiNodeSession]
	UiNodes            []UiNodeSession // Transient field - only filled forthe time of template execution
	LogListRequest     *NodeRequest    // Outstanding request for log list
	LogList            []string
	LogListError       string
}

type SessionHandler struct {
	nodeSessionsLock sync.Mutex
	nodeSessions     map[uint64]*NodeSession
	uiSessionsLock   sync.Mutex
	uiSessions       map[string]*UiSession
	uiTemplate       *template.Template
}

func (sh *SessionHandler) allocateNewNodeSession() (uint64, *NodeSession) {
	sh.nodeSessionsLock.Lock()
	defer sh.nodeSessionsLock.Unlock()
	pin := uint64(weakrand.Int63n(100_000_000))
	for _, ok := sh.nodeSessions[pin]; ok; _, ok = sh.nodeSessions[pin] {
		pin = uint64(weakrand.Int63n(100_000_000))
	}
	nodeSession := &NodeSession{requestCh: make(chan *NodeRequest, 16)}
	sh.nodeSessions[pin] = nodeSession
	return pin, nodeSession
}

func (sh *SessionHandler) findNodeSession(pin uint64) (*NodeSession, bool) {
	sh.nodeSessionsLock.Lock()
	defer sh.nodeSessionsLock.Unlock()
	nodeSession, ok := sh.nodeSessions[pin]
	return nodeSession, ok
}

func (sh *SessionHandler) newUiSession() (string, *UiSession, error) {
	var b [32]byte
	var sessionId string
	_, err := io.ReadFull(rand.Reader, b[:])
	if err == nil {
		sessionId = base64.URLEncoding.EncodeToString(b[:])
	}
	uiSession := &UiSession{uiNodeTree: btree.NewG[UiNodeSession](32, func(a, b UiNodeSession) bool {
		return strings.Compare(a.SessionName, b.SessionName) < 0
	})}
	sh.uiSessionsLock.Lock()
	defer sh.uiSessionsLock.Unlock()
	if sessionId != "" {
		sh.uiSessions[sessionId] = uiSession
	}
	return sessionId, uiSession, err
}

func (sh *SessionHandler) findUiSession(sessionId string) (*UiSession, bool) {
	sh.uiSessionsLock.Lock()
	defer sh.uiSessionsLock.Unlock()
	uiSession, ok := sh.uiSessions[sessionId]
	return uiSession, ok
}

const resumeOperatorSessionName = "resume_session"
const sessionIdCookieName = "sessionId"
const sessionIdCookieDuration = 30 * 24 * 3600 // 30 days

var uiRegex = regexp.MustCompile("^/ui/(cmdline|)$")

func (sh *SessionHandler) validSessionName(sessionName string, uiSession *UiSession) bool {
	if sessionName == "" {
		uiSession.Errors = append(uiSession.Errors, "empty session name")
		return false
	}
	if uiSession.uiNodeTree.Has(UiNodeSession{SessionName: sessionName}) {
		uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("session with name [%s] already present, choose another name or close [%s]", sessionName, sessionName))
		return false
	}
	return true
}

func (sh *SessionHandler) fetchCmdLineArgs(w http.ResponseWriter, r *http.Request, uiSession *UiSession) {
	rbuf := bufio.NewReaderSize(r.Body, 128 /* Maximum length of the sessionName */)
	line, isPrefix, err := rbuf.ReadLine()
	if err != nil {
		fmt.Fprintf(w, "ERROR: Reading sessionName: %v\n", err)
		return
	}
	if isPrefix {
		fmt.Fprintf(w, "ERROR: Session name is too long\n")
		return
	}
	sessionName := string(line)
	if sessionName == "" {
		fmt.Fprintf(w, "ERROR: Empty session name\n")
		return
	}
	if v, ok := uiSession.uiNodeTree.Get(UiNodeSession{SessionName: sessionName}); ok {
		if uiSession.NodeS, ok = sh.findNodeSession(v.SessionPin); !ok {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
			uiSession.uiNodeTree.Delete(v)
		} else {
			uiSession.Session = true
			uiSession.SessionName = sessionName
			uiSession.SessionPin = v.SessionPin
		}
	}
	if uiSession.NodeS == nil {
		fmt.Fprintf(w, "ERROR: Node is not allocated\n")
		return
	}
	// Request command line arguments
	nodeRequest := &NodeRequest{url: "/cmdline\n"}
	uiSession.NodeS.requestCh <- nodeRequest
	for nodeRequest != nil {
		nodeRequest.lock.Lock()
		clear := nodeRequest.served
		if nodeRequest.served {
			if nodeRequest.err == "" {
				w.Write(nodeRequest.response)
			} else {
				fmt.Fprintf(w, "ERROR: %s\n", nodeRequest.err)
				if nodeRequest.retries < 16 {
					clear = false
				}
			}
		}
		nodeRequest.lock.Unlock()
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (sh *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := uiRegex.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.NotFound(w, r)
		return
	}
	cookie, err := r.Cookie(sessionIdCookieName)
	var sessionId string
	var uiSession *UiSession
	var sessionFound bool
	if err == nil && cookie.Value != "" {
		sessionId, err = url.QueryUnescape(cookie.Value)
		if err == nil {
			uiSession, sessionFound = sh.findUiSession(sessionId)
		}
	}
	if !sessionFound {
		var e error
		sessionId, uiSession, e = sh.newUiSession()
		if e == nil {
			cookie := http.Cookie{Name: sessionIdCookieName, Value: url.QueryEscape(sessionId), Path: "/", HttpOnly: true, MaxAge: sessionIdCookieDuration}
			http.SetCookie(w, &cookie)
		} else {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Creating new UI session: %v", e))
		}
	}
	uiSession.lock.Lock()
	defer func() {
		uiSession.Session = false
		uiSession.Errors = nil
		uiSession.NodeS = nil
		uiSession.UiNodes = nil
		uiSession.lock.Unlock()
	}()
	if err != nil {
		uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Cookie handling: %v", err))
	}
	switch m[1] {
	case "cmdline":
		sh.fetchCmdLineArgs(w, r, uiSession)
		return
	}
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Parsing form: %v", err)
		return
	}
	if !uiSession.Session {
		// Try to lookup current session name
		currentSessionName := r.FormValue("current_sessionname")
		if currentSessionName != "" {
			if v, ok := uiSession.uiNodeTree.Get(UiNodeSession{SessionName: currentSessionName}); ok {
				if uiSession.NodeS, ok = sh.findNodeSession(v.SessionPin); !ok {
					uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
					uiSession.uiNodeTree.Delete(v)
				} else {
					uiSession.Session = true
					uiSession.SessionName = currentSessionName
					uiSession.SessionPin = v.SessionPin
				}
			}
		}
	}
	// Fill out LogList or LogListError
	if uiSession.LogListRequest != nil {
		uiSession.LogListRequest.lock.Lock()
		clear := uiSession.LogListRequest.served
		if uiSession.LogListRequest.served {
			if uiSession.LogListRequest.err == "" {
				list := strings.Split(string(uiSession.LogListRequest.response), "\n")
				if len(list) > 0 && list[0] == "SUCCESS" {
					list = list[1:]
				}
				uiSession.LogList = list
				uiSession.LogListError = ""
			} else {
				uiSession.LogList = nil
				uiSession.LogListError = uiSession.LogListRequest.err
				if uiSession.LogListRequest.retries < 16 {
					clear = false
				}
			}
		}
		uiSession.LogListRequest.lock.Unlock()
		if clear {
			uiSession.LogListRequest = nil
		}
	}
	sessionName := r.FormValue("sessionname")
	switch {
	case r.FormValue("new_session") != "":
		// Generate new node session PIN that does not exist yet
		if !sh.validSessionName(sessionName, uiSession) {
			break
		}
		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin, uiSession.NodeS = sh.allocateNewNodeSession()
		uiSession.uiNodeTree.ReplaceOrInsert(UiNodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})
	case r.FormValue("resume_session") != "":
		// Resume (take over) node session using known PIN
		pinStr := r.FormValue("pin")
		sessionPin, err := strconv.ParseUint(pinStr, 10, 64)
		if err != nil {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Parsing session PIN %s: %v", pinStr, err))
			break
		}
		var ok bool
		if uiSession.NodeS, ok = sh.findNodeSession(sessionPin); !ok {
			uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", sessionPin))
			break
		}
		if !sh.validSessionName(sessionName, uiSession) {
			break
		}
		uiSession.Session = true
		uiSession.SessionName = sessionName
		uiSession.SessionPin = sessionPin
		uiSession.uiNodeTree.ReplaceOrInsert(UiNodeSession{SessionName: sessionName, SessionPin: uiSession.SessionPin})
	case r.FormValue("log_list") != "":
		// Request list of logs
		if uiSession.NodeS != nil && uiSession.LogListRequest == nil {
			nodeRequest := &NodeRequest{url: "/logs/list\n"}
			uiSession.NodeS.requestCh <- nodeRequest
			uiSession.LogListRequest = nodeRequest
		}
	default:
		// Make one of the previously known sessions active
		for k, vs := range r.Form {
			if len(vs) == 1 {
				if v, ok := uiSession.uiNodeTree.Get(UiNodeSession{SessionName: vs[0]}); ok && fmt.Sprintf("pin%d", v.SessionPin) == k {
					if uiSession.NodeS, ok = sh.findNodeSession(v.SessionPin); !ok {
						uiSession.Errors = append(uiSession.Errors, fmt.Sprintf("Session %d is not found", v.SessionPin))
						uiSession.uiNodeTree.Delete(v)
						break
					}
					uiSession.Session = true
					uiSession.SessionName = vs[0]
					uiSession.SessionPin = v.SessionPin
				}
			}
		}
	}
	// Populate transient field UiNodes to display the buttons (with the labels)
	uiSession.uiNodeTree.Ascend(func(uiNodeSession UiNodeSession) bool {
		uiSession.UiNodes = append(uiSession.UiNodes, uiNodeSession)
		return true
	})
	if err := sh.uiTemplate.Execute(w, uiSession); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		return
	}
}

func webServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := http.NewServeMux()
	sessionTemplate, err := template.ParseFS(assets.Templates, "template/session.html")
	if err != nil {
		return fmt.Errorf("parsing session.html template: %v", err)
	}
	sh := &SessionHandler{
		nodeSessions: map[uint64]*NodeSession{},
		uiSessions:   map[string]*UiSession{},
		uiTemplate:   sessionTemplate,
	}
	mux.Handle("/script/", http.FileServer(http.FS(assets.Scripts)))
	mux.Handle("/ui/", sh)
	bh := &BridgeHandler{sh: sh}
	mux.Handle("/support/", bh)
	certPool := x509.NewCertPool()
	for _, caCertFile := range caCertFiles {
		caCert, err := ioutil.ReadFile(caCertFile)
		if err != nil {
			return fmt.Errorf("reading server certificate: %v", err)
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
	if err = s.ListenAndServeTLS(serverCertFile, serverKeyFile); err != nil {
		return fmt.Errorf("running server: %v", err)
	}
	return nil
}
