package main

import (
	"github.com/dns"
	"strconv"
	"os"
	"path/filepath"
	"bufio"
	"strings"
	"github.com/golang/glog"
)

type Job struct {
	w *dns.ResponseWriter
	r *dns.Msg
}

func worker(id int, jobs <-chan Job, wfJob chan<- *string) {
	glog.Infoln("start work(", id, ").")
	for j := range jobs {
		doWork(j, wfJob)
	}
}

func flushDNSEndpoints(jobs chan *string) {
	filePath := filepath.Join(cf.DnsCacheDir, "dns.cache.file")
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		glog.Infoln("flushDNSEndpoints error")
	}()

	var answer interface{}
	for endpoint := range jobs {
		answer, _ = dnsAnswerMap.Load(*endpoint)
		if answer == nil || !(answer.(DNSAnswer).syncedFile) {
			glog.Infof("write file: %s\n", *endpoint)
			if _, err = f.WriteString(*endpoint + "\n"); err != nil {
				panic(err)
			}
		}
	}
}

func Start(workerSize int, queueSize int) chan<- Job {
	loadEndpointsFromFile()

	jobs := make(chan Job, queueSize)
	wfJobs := make(chan *string, 1000)
	go func() {
		flushDNSEndpoints(wfJobs)
	}()
	for i := 0; i < workerSize; i++ {
		go worker(i, jobs, wfJobs)
	}
	glog.Infoln("Start() done")
	return jobs
}

func loadEndpointsFromFile() {
	cf, _ := os.Open(filepath.Join(cf.DnsCacheDir, "dns.cache.file"))
	defer cf.Close()
	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		arr := strings.Split(scanner.Text(), ";")
		t, _ := strconv.Atoi(arr[1])
		baseLookup(arr[0], uint16(t), nil, nil)
	}
	glog.Infoln("flushDNSEndpoints() done")
}

func doWork(job Job, wfJob chan<- *string) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	msg := dns.Msg{}
	msg.SetReply(job.r)

	domain := msg.Question[0].Name
	dnsType := job.r.Question[0].Qtype
	switch dnsType {
	case dns.TypeA:
		msg.Authoritative = true
		lookup(domain, &msg, dns.TypeA, wfJob)
	default:
		lookup(domain, &msg, dnsType, wfJob)
	}
	(*job.w).WriteMsg(&msg)
}
