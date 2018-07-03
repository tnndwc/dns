package main

import (
	"github.com/dns"
	"time"
	"net"
	"strconv"
	"github.com/golang/glog"
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
			glog.Errorln(err)
		}
	}()
	key := domain + ";" + strconv.Itoa(int(dnsType))
	answer, _ := dnsAnswerMap.Load(key)
	syncedFile := false
	if answer != nil {
		syncedFile = true
		dnsAnswer := answer.(DNSAnswer)
		duration := time.Since(dnsAnswer.refreshTime)
		glog.Infoln(duration.Minutes())
		if duration.Minutes() < 10 {
			glog.Infoln("Load answer from cache")
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
		glog.Warningf("error: %s\n", err.Error())
		return nil, nil
	}
	if r.Rcode != dns.RcodeSuccess {
		glog.Warningf(" invalid answer name %s after MX query for %s\n", domain, domain)
		if r.Answer == nil || len(r.Answer) <= 0 {
			return nil, nil
		}
	}

	glog.Infoln(key + " -> Store answer to cache ")
	dnsAnswerMap.Store(key, DNSAnswer{r.Answer, syncedFile, time.Now()})

	if syncedFile {
		return r, nil
	} else {
		return r, &key
	}
}
