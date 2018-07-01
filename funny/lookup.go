package main

import (
	"fmt"
	"github.com/dns"
	"time"
	"net"
	"strconv"
	"log"
)

type DNSAnswer struct {
	answer      []dns.RR
	syncedFile  bool
	refreshTime time.Time
}

func lookup(domain string, msg *dns.Msg, dnsType uint16, wfJob chan<- *string) {
	r, key := baseLookup(domain, dnsType, msg)
	if r != nil {
		msg.Answer = r.Answer
		if key != nil {
			wfJob <- key
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
	syncedFile := false
	if answer != nil {
		syncedFile = true
		dnsAnswer := answer.(DNSAnswer)
		duration := time.Since(dnsAnswer.refreshTime)
		fmt.Println(duration.Minutes())
		if duration.Minutes() < 1 {
			fmt.Println("Load answer from cache")
			if msg != nil {
				msg.Answer = dnsAnswer.answer
			}
			return nil, nil
		}
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
	dnsAnswerMap.Store(key, DNSAnswer{r.Answer, syncedFile, time.Now()})

	if syncedFile {
		return r, nil
	} else {
		return r, &key
	}
}
