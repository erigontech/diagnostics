package cmd

import (
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
	"mime"
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
		http.Error(w, "Error parsing session PIN", http.StatusBadRequest)
		log.Printf("Errir parsing session pin %s: %v\n", m[1], err)
		return
	}
	nodeSession, ok := bh.sh.findNodeSession(pin)
	if !ok {
		http.Error(w, fmt.Sprintf("Session with specified PIN %d not found", pin), http.StatusBadRequest)
		log.Printf("Session with specified PIN %d not found\n", pin)
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

	var versionBytes [8]byte
	if _, err := io.ReadFull(r.Body, versionBytes[:]); err != nil {
		http.Error(w, fmt.Sprintf("Error reading version bytes: %v", err), http.StatusBadRequest)
		log.Printf("Error reading version bytes: %v\n", err)
		return
	}
	nodeSession.SupportVersion = binary.BigEndian.Uint64(versionBytes[:])

	var writeBuf bytes.Buffer
	for request := range nodeSession.requestCh {
		request.lock.Lock()
		url := request.url
		request.lock.Unlock()
		fmt.Printf("Sending request %s\n", url)
		writeBuf.Reset()
		fmt.Fprintf(&writeBuf, url)
		if _, err := w.Write(writeBuf.Bytes()); err != nil {
			log.Printf("Writing metrics request: %v\n", err)
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
			log.Printf("Reading size of metrics response: %v\n", err)
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
			log.Printf("Reading metrics response: %v\n", err)
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
	lock           sync.Mutex
	sessionPin     uint64
	Connected      bool
	RemoteAddr     string
	SupportVersion uint64            // Version of the erigon support command
	requestCh      chan *NodeRequest // Channel for incoming metrics requests
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
}

func (uiSession *UiSession) appendError(err string) {
	uiSession.lock.Lock()
	defer uiSession.lock.Unlock()
	uiSession.Errors = append(uiSession.Errors, err)
}

type Versions struct {
	Success        bool
	Error          string
	NodeVersion    uint64
	SupportVersion uint64
	CodeVersion    string
	GitCommit      string
}
type CmdLineArgs struct {
	Success bool
	Error   string
	Args    string
}
type LogListItem struct {
	Filename    string
	Size        int64
	PrintedSize string
}
type LogList struct {
	Success     bool
	Error       string
	SessionName string
	List        []LogListItem
}
type LogPart struct {
	Success bool
	Error   string
	Lines   []string
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

func (sh *SessionHandler) fetch(url string, requestChannel chan *NodeRequest) (bool, string) {
	if requestChannel == nil {
		return false, fmt.Sprintf("ERROR: Node is not allocated\n")
	}
	// Request command line arguments
	nodeRequest := &NodeRequest{url: url}
	requestChannel <- nodeRequest
	var sb strings.Builder
	var success bool
	for nodeRequest != nil {
		nodeRequest.lock.Lock()
		clear := nodeRequest.served
		if nodeRequest.served {
			if nodeRequest.err == "" {
				sb.Reset()
				sb.Write(nodeRequest.response)
				success = true
			} else {
				success = false
				fmt.Fprintf(&sb, "ERROR: %s\n", nodeRequest.err)
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
	return success, sb.String()
}

const successLine = "SUCCESS"

func (sh *SessionHandler) processVersions(w http.ResponseWriter, success bool, result string) {
	var versions Versions
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
			versions.Success = true
			if len(lines) < 2 {
				versions.Error = fmt.Sprintf("at least node version needs to be present")
				versions.Success = false
			} else {
				var err error
				versions.NodeVersion, err = strconv.ParseUint(lines[1], 10, 64)
				if err != nil {
					versions.Error = fmt.Sprintf("parsing node version [%s]: %v", lines[1], err)
					versions.Success = false
				} else {
					for idx, line := range lines[2:] {
						switch idx {
						case 0:
							versions.CodeVersion = line
						case 1:
							versions.GitCommit = line
						}
					}
				}
			}
		} else {
			versions.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
		}
	} else {
		versions.Error = result
	}
	if err := sh.uiTemplate.ExecuteTemplate(w, "versions.html", versions); err != nil {
		fmt.Fprintf(w, "Executing versions template: %v", err)
		return
	}
}

func (sh *SessionHandler) processCmdLineArgs(w http.ResponseWriter, success bool, result string) {
	var args CmdLineArgs
	if success {
		if strings.HasPrefix(result, successLine) {
			args.Args = strings.ReplaceAll(result[len(successLine):], "\n", " ")
		} else {
			args.Args = result
		}
		args.Success = true
	} else {
		args.Success = false
		args.Error = result
	}
	if err := sh.uiTemplate.ExecuteTemplate(w, "cmd_line.html", args); err != nil {
		fmt.Fprintf(w, "Executing cmd_line template: %v", err)
	}
}

func MBToGB(b uint64) (float64, int) {
	const unit = 1024
	if b < unit {
		return float64(b), 0
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return float64(b) / float64(div), exp
}

func ByteCount(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	bGb, exp := MBToGB(b)
	return fmt.Sprintf("%.1f%cB", bGb, "KMGTPE"[exp])
}

func (sh *SessionHandler) processLogList(w http.ResponseWriter, success bool, sessionName string, result string) {
	var list = LogList{SessionName: sessionName}
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
			list.Success = true
			for _, line := range lines[1:] {
				if len(line) == 0 {
					// skip empty line (usually at the end)
					continue
				}
				terms := strings.Split(line, " | ")
				if len(terms) != 2 {
					list.Error = fmt.Sprintf("incorrect response line (need to have 2 terms divided by |): %v", line)
					list.Success = false
					break
				}
				size, err := strconv.ParseUint(terms[1], 10, 64)
				if err != nil {
					list.Error = fmt.Sprintf("incorrect size: %v", terms[1])
					list.Success = false
					break
				}
				list.List = append(list.List, LogListItem{Filename: terms[0], Size: int64(size), PrintedSize: ByteCount(size)})
			}
		} else {
			list.Error = fmt.Sprintf("incorrect response (first line needs to be SUCCESS): %v", lines)
		}
	} else {
		list.Error = result
	}
	if err := sh.uiTemplate.ExecuteTemplate(w, "log_list.html", list); err != nil {
		fmt.Fprintf(w, "Executing log_list template: %v", err)
		return
	}
}

func (sh *SessionHandler) processLogPart(w http.ResponseWriter, success bool, sessionName string, result string) {
	var part LogPart
	if success {
		lines := strings.Split(result, "\n")
		if len(lines) > 0 && strings.HasPrefix(lines[0], successLine) {
			part.Lines = lines[1:]
		} else {
			part.Lines = lines
		}
		part.Success = true
	} else {
		part.Success = false
		part.Error = result
	}
	if err := sh.uiTemplate.ExecuteTemplate(w, "log_read.html", part); err != nil {
		fmt.Fprintf(w, "Executing log_read template: %v", err)
		return
	}
}

func (sh *SessionHandler) lookupSession(r *http.Request, uiSession *UiSession) chan *NodeRequest {
	uiSession.lock.Lock()
	defer uiSession.lock.Unlock()
	uiSession.NodeS = nil
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
	if uiSession.NodeS != nil {
		return uiSession.NodeS.requestCh
	}
	return nil
}

var logReadFirstLine = regexp.MustCompile("^SUCCESS: ([0-9]+)-([0-9]+)/([0-9]+)$")

func parseLogPart(nodeRequest *NodeRequest, offset uint64) (bool, uint64, uint64, []byte, string) {
	nodeRequest.lock.Lock()
	defer nodeRequest.lock.Unlock()
	if !nodeRequest.served {
		return false, 0, 0, nil, ""
	}
	clear := nodeRequest.retries >= 16
	if nodeRequest.err != "" {
		return clear, 0, 0, nil, nodeRequest.err
	}
	firstLineEnd := bytes.IndexByte(nodeRequest.response, '\n')
	if firstLineEnd == -1 {
		return clear, 0, 0, nil, fmt.Sprintf("could not find first line in log part response")
	}
	m := logReadFirstLine.FindSubmatch(nodeRequest.response[:firstLineEnd])
	if m == nil {
		return clear, 0, 0, nil, fmt.Sprintf("first line needs to have format SUCCESS: from-to/total, was [%sn", nodeRequest.response[:firstLineEnd])
	}
	from, err := strconv.ParseUint(string(m[1]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing from: %v", err)
	}
	if from != offset {
		return clear, 0, 0, nil, fmt.Sprintf("Unexpected from %d, wanted %d", from, offset)
	}
	to, err := strconv.ParseUint(string(m[2]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing to: %v", err)
	}
	total, err := strconv.ParseUint(string(m[3]), 10, 64)
	if err != nil {
		return clear, 0, 0, nil, fmt.Sprintf("parsing total: %v", err)
	}
	return true, to, total, nodeRequest.response[firstLineEnd+1:], ""
}

type LogReader struct {
	filename       string
	requestChannel chan *NodeRequest
	total          uint64
	offset         uint64
	ctx            context.Context
}

func (lr *LogReader) Read(p []byte) (n int, err error) {
	nodeRequest := &NodeRequest{url: fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(lr.filename), lr.offset)}
	lr.requestChannel <- nodeRequest
	var total uint64
	var clear bool
	var part []byte
	var errStr string
	for nodeRequest != nil {
		select {
		case <-lr.ctx.Done():
			return 0, fmt.Errorf("interrupted")
		default:
		}
		clear, _, total, part, errStr = parseLogPart(nodeRequest, lr.offset)
		if clear {
			nodeRequest = nil
		} else {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if errStr != "" {
		return 0, fmt.Errorf(errStr)
	}
	lr.total = total
	copied := copy(p, part)
	lr.offset += uint64(copied)
	if lr.offset == total {
		return copied, io.EOF
	}
	return copied, nil
}

func (lr *LogReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		lr.offset = uint64(offset)
	case io.SeekCurrent:
		lr.offset = uint64(int64(lr.offset) + offset)
	case io.SeekEnd:
		if lr.total > 0 {
			lr.offset = uint64(int64(lr.total) + offset)
		} else {
			lr.offset = 0
		}
	}
	return int64(lr.offset), nil
}

func transmitLogFile(ctx context.Context, r *http.Request, w http.ResponseWriter, sessionName string, filename string, size uint64, requestChannel chan *NodeRequest) {
	if requestChannel == nil {
		fmt.Fprintf(w, "ERROR: Node is not allocated\n")
		return
	}
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": sessionName + "_" + filename})
	w.Header().Set("Content-Disposition", cd)
	w.Header().Set("Content-Type", "application/octet-stream")
	logReader := &LogReader{filename: filename, requestChannel: requestChannel, offset: 0, total: size, ctx: ctx}
	http.ServeContent(w, r, filename, time.Now(), logReader)
}

var uiRegex = regexp.MustCompile("^/ui/(cmd_line|log_list|log_head|log_tail|log_download|versions|)$")

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
			uiSession.appendError(fmt.Sprintf("Creating new UI session: %v", e))
		}
	}
	if err != nil {
		uiSession.appendError(fmt.Sprintf("Cookie handling: %v", err))
	}
	// Try to lookup current session name
	if err := r.ParseForm(); err != nil {
		fmt.Fprintf(w, "Parsing form: %v", err)
		return
	}
	requestChannel := sh.lookupSession(r, uiSession)
	filename := r.Form.Get("file")
	sizeStr := r.Form.Get("size")
	sessionName := r.Form.Get("current_sessionname")
	switch m[1] {
	case "versions":
		success, result := sh.fetch("/version\n", requestChannel)
		sh.processVersions(w, success, result)
		return
	case "cmd_line":
		success, result := sh.fetch("/cmdline\n", requestChannel)
		sh.processCmdLineArgs(w, success, result)
		return
	case "log_list":
		success, result := sh.fetch("/logs/list\n", requestChannel)
		sh.processLogList(w, success, uiSession.SessionName, result)
		return
	case "log_head":
		success, result := sh.fetch(fmt.Sprintf("/logs/read?file=%s&offset=0\n", url.QueryEscape(filename)), requestChannel)
		sh.processLogPart(w, success, uiSession.SessionName, result)
		return
	case "log_tail":
		size, err := strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Parsing size %s: %v", sizeStr, err)
			return
		}
		var offset uint64
		if size > 16*1024 {
			offset = size - 16*1024
		}
		success, result := sh.fetch(fmt.Sprintf("/logs/read?file=%s&offset=%d\n", url.QueryEscape(filename), offset), requestChannel)
		sh.processLogPart(w, success, uiSession.SessionName, result)
		return
	case "log_download":
		size, err := strconv.ParseUint(sizeStr, 10, 64)
		if err != nil {
			fmt.Fprintf(w, "Parsing size %s: %v", sizeStr, err)
			return
		}
		transmitLogFile(r.Context(), r, w, sessionName, filename, size, requestChannel)
		return
	}
	uiSession.lock.Lock()
	defer func() {
		uiSession.Session = false
		uiSession.Errors = nil
		uiSession.NodeS = nil
		uiSession.UiNodes = nil
		uiSession.lock.Unlock()
	}()
	sessionName = r.FormValue("sessionname")
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
	if err := sh.uiTemplate.ExecuteTemplate(w, "session.html", uiSession); err != nil {
		fmt.Fprintf(w, "Executing template: %v", err)
		return
	}
}

func webServer() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mux := http.NewServeMux()
	uiTemplate, err := template.ParseFS(assets.Templates, "template/*.html")
	if err != nil {
		return fmt.Errorf("parsing session.html template: %v", err)
	}
	sh := &SessionHandler{
		nodeSessions: map[uint64]*NodeSession{},
		uiSessions:   map[string]*UiSession{},
		uiTemplate:   uiTemplate,
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
		select {
		case <-ctx.Done():
			return nil
		default:
			return fmt.Errorf("running server: %v", err)
		}
	}
	return nil
}
