package main

import (
	"github.com/dns"
	"fmt"
	"strconv"
	"time"
	"net"
	"log"
	"os"
	"path/filepath"
	"bufio"
	"strings"
)

type Job struct {
	w *dns.ResponseWriter
	r *dns.Msg
}

func worker(id int, jobs <-chan Job, wfJob chan<- string) {
	for j := range jobs {
		doWork(j, wfJob)
	}
}

func writeFileWorker(jobs chan string) {
	filePath := filepath.Join(*confFolder, "dns.file")
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		log.Println("writeFileWorker error")
	}()
	for endpoint := range jobs {
		log.Println(fmt.Printf("write file: %s\n", endpoint))
		if _, err = f.WriteString(endpoint + "\n"); err != nil {
			panic(err)
		}
	}
}

func Start(workerSize int, queueSize int) chan<- Job {
	go func() {
		loadEndpointsFromFile()
	}()

	jobs := make(chan Job, queueSize)
	wfJobs := make(chan string, 1000)
	go func() {
		writeFileWorker(wfJobs)
	}()
	for i := 0; i < workerSize; i++ {
		go worker(i, jobs, wfJobs)
	}
	return jobs
}

func loadEndpointsFromFile() {
	cf, _ := os.Open(filepath.Join(*confFolder, "dns.file"))
	defer cf.Close()
	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		arr := strings.Split(scanner.Text(), ";")
		t, _ := strconv.Atoi(arr[1])
		baseLookup(arr[0], uint16(t), nil)
	}
	log.Println("--------writeFileWorker() done")
}

func doWork(job Job, wfJob chan<- string) {
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
	switch job.r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		lookup(domain, &msg, dns.TypeA, wfJob)
	default:
		lookup(domain, &msg, job.r.Question[0].Qtype, wfJob)
	}
	(*job.w).WriteMsg(&msg)
}

func lookup(domain string, msg *dns.Msg, dnsType uint16, wfJob chan<- string) {
	r, key := baseLookup(domain, dnsType, msg)
	if r != nil {
		msg.Answer = r.Answer
		if key != nil {
			wfJob <- *key
		}
	}
}

func baseLookup(domain string, dnsType uint16, msg *dns.Msg) (*dns.Msg, *string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	key := domain + ";" + strconv.Itoa(int(dnsType))
	answer, _ := dnsAnswerMap.Load(key)

	if answer != nil {
		fmt.Println("Load answer from cache")
		if msg != nil {
			msg.Answer = answer.([]dns.RR)
		}
		return nil, nil
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dnsType)
	m.RecursionDesired = true

	var c = new(dns.Client)

	c.ReadTimeout = time.Duration(10) * 1e9
	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		log.Println(fmt.Printf("error: %s\n", err.Error()))
		return nil, nil
	}
	if r.Rcode != dns.RcodeSuccess {
		log.Println(fmt.Printf(" invalid answer name %s after MX query for %s\n", domain, domain))
	}
	// Stuff must be in the answer section
	/*for _, a := range r.Answer {
		fmt.Printf("%v\n", a)
	}*/

	fmt.Println(key + " -> Store answer to cache")
	dnsAnswerMap.Store(key, r.Answer)
	return r, &key
}
