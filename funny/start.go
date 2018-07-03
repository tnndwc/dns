package main

import (
	"log"
	"github.com/dns"
	"sync"
	"flag"
	"path/filepath"
)

var dnsAnswerMap sync.Map

var config *dns.ClientConfig

type handler struct{}

var workers chan<- Job

var confFolder *string

func (hd *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	workers <- Job{&w, r}
}

func main() {
	confFolder = flag.String("cf", "", "Conf folder")
	dnsPort := flag.String("p", "", "port")
	dnsHost := flag.String("h", "", "host")
	flag.Parse()
	config, _ = dns.ClientConfigFromFile(filepath.Join(*confFolder, "dns.conf"))
	workers = Start(10, 15)
	srv := &dns.Server{Addr: *dnsHost + ":" + *dnsPort, Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
