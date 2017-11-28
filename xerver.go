// xerver 3.0, a tiny and light transparent fastcgi reverse proxy,
// copyright 2016, (c) Mohammed Al Ashaal <http://www.alash3al.xyz>,
// published uner MIT licnese .

// https://github.com/alash3al/xerver

// -----------------------------
// *> available options
// >> --root        [only use xerver as static file server],            i.e "/var/www/" .
// >> --backend     [only use xerver as fastcgi reverse proxy],         i.e "[unix|tcp]:/var/run/php5-fpm.sock" .
// >> --controller  [the fastcgi process main file "SCRIPT_FILENAME"],  i.e "/var/www/main.php"
// >> --http        [the local http address to listen on],              i.e ":80"
// >> --https       [the local https address to listen on],             i.e ":443"
// >> --cert        [the ssl cert file path],                           i.e "/var/ssl/ssl.cert"
// >> --key         [the ssl key file path],                            i.e "/var/ssl/ssl.key"
package main

import "os"
import "io"
import "fmt"
import "log"
import "net"
import "flag"
import "strconv"
import "strings"

import "net/http"

import "github.com/majorcode/fcgi_client"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// user vars
var (
	ROOT       *string = flag.String("root", "", "the static files root directory, (default empty)")
	BACKEND    *string = flag.String("backend", "", "the fastcgi backend address, (default empty)")
	CONTROLLER *string = flag.String("controller", "", "the fastcgi main controller file, (default empty)")
	HTTP       *string = flag.String("http", ":80", "the http-server local address")
	HTTPS      *string = flag.String("https", "", "the https-server local address, (default empty)")
	CERT       *string = flag.String("cert", "", "the ssl cert file, (default empty)")
	KEY        *string = flag.String("key", "", "the ssl key file, (default empty)")
)

// system vars
var (
	softwareVersion = "xerver/v3.1"
	fcgiProtocol    = ""
	fcgiAddress     = ""
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func serveFCGI(res http.ResponseWriter, req *http.Request) {
	// connect to the fastcgi backend,
	// and check whether there is an error or not .
	fcgi, err := fcgiclient.Dial(fcgiProtocol, fcgiAddress)
	if err != nil {
		log.Println(err)
		http.Error(res, "Unable to connect to the backend", 502)
		return
	}
	// automatically close the fastcgi connection and the requested body at the end .

	defer fcgi.Close()
	defer req.Body.Close()
	// prepare some vars :
	// -- http[addr, port]
	// -- https[addr, port]
	// -- remote[addr, host, port]
	// -- edit the request path
	// -- environment variables
	httpAddr, httpPort, _ := net.SplitHostPort(*HTTP)
	httpsAddr, httpsPort, _ := net.SplitHostPort(*HTTPS)
	remoteAddr, remotePort, _ := net.SplitHostPort(req.RemoteAddr)
	req.URL.Path = req.URL.ResolveReference(req.URL).Path

	rURI := req.Header.Get("Request-Uri")
	if rURI == "" {
		rURI = req.URL.RequestURI()
	}

	env := map[string]string{
		"SCRIPT_FILENAME": *CONTROLLER,
		"REQUEST_METHOD":  req.Method,
		"REQUEST_URI":     rURI,
		"REQUEST_PATH":    req.URL.Path,
		"PATH_INFO":       req.URL.Path,
		"CONTENT_LENGTH":  fmt.Sprintf("%d", req.ContentLength),
		"CONTENT_TYPE":    req.Header.Get("Content-Type"),
		"REMOTE_ADDR":     remoteAddr,
		"REMOTE_PORT":     remotePort,
		"REMOTE_HOST":     remoteAddr,
		"QUERY_STRING":    req.URL.Query().Encode(),
		"SERVER_SOFTWARE": softwareVersion,
		"SERVER_NAME":     req.Host,
		"SERVER_ADDR":     httpAddr,
		"SERVER_PORT":     httpPort,
		"SERVER_PROTOCOL": req.Proto,
		"FCGI_PROTOCOL":   fcgiProtocol,
		"FCGI_ADDR":       fcgiAddress,
		"HTTPS":           "off",
		"HTTP_HOST":       req.Host,
	}
	// tell fastcgi backend that, this connection is done over https connection if enabled .
	if req.TLS != nil {
		env["HTTPS"] = "on"
		env["SERVER_PORT"] = httpsPort
		env["SERVER_ADDR"] = httpsAddr
		env["SSL_CERT"] = *CERT
		env["SSL_KEY"] = *KEY
	}
	// iterate over request headers and append them to the environment varibales in the valid format .
	for k, v := range req.Header {
		if k == "Request-Uri" {
			continue
		}
		env["HTTP_"+strings.Replace(strings.ToUpper(k), "-", "_", -1)] = strings.Join(v, ";")
	}
	// fethcing the response from the fastcgi backend,
	// and check for errors .
	resp, err := fcgi.Request(env, req.Body)
	if err != nil {
		log.Println("err> ", err.Error())
		// TODO: This is an ugly plaintext error page on the client
		http.Error(res, "Unable to fetch the response from the backend", 502)
		return
	}
	// parse the fastcgi status .
	resp.Status = resp.Header.Get("Status")
	resp.StatusCode, _ = strconv.Atoi(strings.Split(resp.Status, " ")[0])
	if resp.StatusCode < 100 {
		resp.StatusCode = 200
	}
	// automatically close the fastcgi response body at the end .
	defer resp.Body.Close()
	// read the fastcgi response headers,
	// and apply the actions related to them .
	for k, v := range resp.Header {
		for i := 0; i < len(v); i++ {
			if res.Header().Get(k) == "" {
				res.Header().Set(k, v[i])
			} else {
				res.Header().Add(k, v[i])
			}
		}
	}
	res.Header().Set("Server", softwareVersion)
	// fix the redirect issues by fetching the fastcgi response location header
	// then redirect the client, then ignore any output .
	if resp.Header.Get("Location") != "" {
		http.Redirect(res, req, resp.Header.Get("Location"), resp.StatusCode)
		return
	}
	// write the response status code .
	res.WriteHeader(resp.StatusCode)
	// only sent the header if the request method isn't HEAD .
	if req.Method != "HEAD" {
		io.Copy(res, resp.Body)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// - parse the cmd flags
// - check for the required flags
// - display welcome messages
func init() {
	flag.Parse()
	if (*ROOT == "" && *BACKEND == "") || (*ROOT != "" && *BACKEND != "") {
		log.Fatal("Please, choose whether you want me as transparent static-server or reverse-proxy ?")
	}
	if *ROOT != "" {
		if _, e := os.Stat(*ROOT); e != nil {
			log.Fatal(e)
		}
	}
	if *BACKEND != "" {
		parts := strings.SplitN(*BACKEND, ":", 2)
		if !strings.Contains(*BACKEND, ":") || len(parts) < 2 {
			log.Fatal("Please, provide a valid backend format [protocol:address]")
		}
		fcgiProtocol = parts[0]
		fcgiAddress = parts[1]
		if _, e := os.Stat(*CONTROLLER); e != nil {
			log.Fatal(e)
		}
	}
	if strings.HasPrefix(*HTTP, ":") {
		*HTTP = "0.0.0.0" + *HTTP
	}
	if strings.HasPrefix(*HTTPS, ":") {
		*HTTPS = "0.0.0.0" + *HTTPS
	}

	fmt.Println("Welcome to ", softwareVersion)
	fmt.Println("Backend:           ", *BACKEND)
	fmt.Println("CONTROLLER:        ", *CONTROLLER)
	fmt.Println("HTTP Address:      ", *HTTP)
	fmt.Println("ROOT:              ", *ROOT)
	fmt.Println("HTTPS Address:     ", *HTTPS)
	fmt.Println("SSL Cert:          ", *CERT)
	fmt.Println("SSL Key:           ", *KEY)
	fmt.Println("")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// let's play :)
func main() {
	// handle any panic
	rcvr := func() {
		if err := recover(); err != nil {
			log.Println("err> ", err)
		}
	}
	// the handler
	handler := func(res http.ResponseWriter, req *http.Request) {
		if *ROOT == "" {
			serveFCGI(res, req)
			return
		}
		http.FileServer(http.Dir(*ROOT)).ServeHTTP(res, req)
	}
	// an error channel to catch any error
	err := make(chan error)
	// run a http server in a goroutine
	go (func() {
		defer rcvr()
		err <- http.ListenAndServe(*HTTP, http.HandlerFunc(handler))
	})()
	// run a https server in another goroutine
	go (func() {
		if *HTTPS != "" && *CERT != "" && *KEY != "" {
			defer rcvr()
			err <- http.ListenAndServeTLS(*HTTPS, *CERT, *KEY, http.HandlerFunc(handler))
		}
	})()
	// there is an error occurred,
	// let's catch it, then exit .
	log.Fatal(<-err)
}
