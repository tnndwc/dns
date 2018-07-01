package main

import (
	"github.com/dns"
	"strconv"
	"os"
	"log"
	"path/filepath"
	"bufio"
	"strings"
	"fmt"
)

type Job struct {
	w *dns.ResponseWriter
	r *dns.Msg
}

func worker(id int, jobs <-chan Job, wfJob chan<- *string) {
	log.Println("start work(", id, ").")
	for j := range jobs {
		doWork(j, wfJob)
	}
}

func flushDNSEndpoints(jobs chan *string) {
	filePath := filepath.Join(*confFolder, "dns.cache.file")
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		log.Println("flushDNSEndpoints error")
	}()

	var answer interface{}
	for endpoint := range jobs {
		answer, _ = dnsAnswerMap.Load(*endpoint)
		if answer == nil || !(answer.(DNSAnswer).syncedFile) {
			log.Println(fmt.Printf("write file: %s\n", *endpoint))
			if _, err = f.WriteString(*endpoint + "\n"); err != nil {
				panic(err)
			}
		}
	}
}

func Start(workerSize int, queueSize int) chan<- Job {
	go func() {
		loadEndpointsFromFile()
	}()

	jobs := make(chan Job, queueSize)
	wfJobs := make(chan *string, 1000)
	go func() {
		flushDNSEndpoints(wfJobs)
	}()
	for i := 0; i < workerSize; i++ {
		go worker(i, jobs, wfJobs)
	}
	log.Println("Start() done")
	return jobs
}

func loadEndpointsFromFile() {
	cf, _ := os.Open(filepath.Join(*confFolder, "dns.cache.file"))
	defer cf.Close()
	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		arr := strings.Split(scanner.Text(), ";")
		t, _ := strconv.Atoi(arr[1])
		_, key := baseLookup(arr[0], uint16(t), nil)
		answer, _ := dnsAnswerMap.Load(*key)
		dNSAnswer := answer.(DNSAnswer)
		dNSAnswer.syncedFile = true
		dnsAnswerMap.Store(*key, dNSAnswer)
		answer, _ = dnsAnswerMap.Load(*key)
		fmt.Println(answer)
	}
	log.Println("flushDNSEndpoints() done")
}

func doWork(job Job, wfJob chan<- *string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	msg := dns.Msg{}
	msg.SetReply(job.r)

	/*for _, e := range r.Question {
		fmt.Println(e.String())
	}*/

	//fmt.Println("dns.Type: " + strconv.Itoa(int(r.Question[0].Qtype)))

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
