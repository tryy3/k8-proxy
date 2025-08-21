// This program demonstrates how to use tsnet as a library.
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"path/filepath"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"tailscale.com/tsnet"
)

var (
	addr           = flag.String("addr", ":80", "address to listen on")
	kubeConfigPath *string
)

func main() {
	if home := homedir.HomeDir(); home != "" {
		kubeConfigPath = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeConfigPath = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()
	log.Println("kubeConfigPath", *kubeConfigPath)
	authKey := getAuthKey()

	srv := new(tsnet.Server)
	srv.AuthKey = authKey

	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err.Error())
	}

	proxyService := NewProxyService(srv, clientset)
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
