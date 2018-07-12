package main

import (
	"github.com/dns"
	"sync"
	"flag"
	"github.com/golang/glog"
	"encoding/json"
	"os"
)

var cf Conf

var dnsAnswerMap sync.Map

var config *dns.ClientConfig

type handler struct{}

var workers chan<- Job

type Conf struct {
	Listen      string `json:"listen"`
	DnsCacheDir string `json:"dnsCacheDir"`
	DNSPath     string `json:"dnsPath"`
}

func (hd *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	workers <- Job{&w, r}
}

func main() {
	confFile := flag.String("f", "", "Conf File")
	flag.Parse()

	if len(*confFile) <= 0 {
		glog.Fatalln("Could not find a config file.")
	}

	if _, err := os.Stat(*confFile); os.IsNotExist(err) {
		glog.Fatalf("File %s does not exist.", &confFile)
	}

	err := json.Unmarshal(ReadFile(confFile), &cf)
	if err != nil {
		panic(err)
	}

	config, err = dns.ClientConfigFromFile(cf.DNSPath)
	if err != nil {
		panic(err)
	}

	workers = Start(20, 50)
	srv := &dns.Server{Addr: cf.Listen, Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		glog.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
