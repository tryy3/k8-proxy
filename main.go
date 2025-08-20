// This program demonstrates how to use tsnet as a library.
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"tailscale.com/tsnet"
)

var (
	addr = flag.String("addr", ":80", "address to listen on")
)

func main() {
	flag.Parse()
	authKey := getAuthKey()

	srv := new(tsnet.Server)
	srv.AuthKey = authKey

	proxyService := NewProxyService(srv)
	proxyService.Start()

	select {}

	// defer srv.Close()
	// ln, err := srv.Listen("tcp", *addr)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer ln.Close()

	// lc, err := srv.LocalClient()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if *addr == ":443" {
	// 	ln = tls.NewListener(ln, &tls.Config{
	// 		GetCertificate: lc.GetCertificate,
	// 	})
	// }

	// remote, err := url.Parse("https://www.whatismyip.com/")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// director := func(req *http.Request) {
	// 	req.URL.Scheme = remote.Scheme
	// 	req.URL.Host = remote.Host
	// }

	// proxy := &httputil.ReverseProxy{
	// 	Director: director,
	// }

	// handler := handler{proxy: proxy}

	// http.Handle("/", handler)
	// // err = http.ListenAndServe(*addr, nil)
	// // if err != nil {
	// // 	log.Fatal(err)
	// // }

	// log.Fatal(http.Serve(ln, nil))
}

func firstLabel(s string) string {
	s, _, _ = strings.Cut(s, ".")
	return s
}

type handler struct {
	proxy *httputil.ReverseProxy
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)

	h.proxy.ServeHTTP(w, r)
}
