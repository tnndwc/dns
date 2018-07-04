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
	baseLookup(domain, dnsType, msg, wfJob)
}

func baseLookup(domain string, dnsType uint16, msg *dns.Msg, wfJob chan<- *string) {
	defer func() {
		if err := recover(); err != nil {
			glog.Errorln(err)
		}
	}()
	key := domain + ";" + strconv.Itoa(int(dnsType))
	answer, _ := dnsAnswerMap.Load(key)
	if answer != nil {
		dnsAnswer := answer.(DNSAnswer)
		duration := time.Since(dnsAnswer.refreshTime)
		glog.Infoln(duration.Minutes())
		if duration.Minutes() < 10 {
			glog.Infof("Load answer from cache: %s  <-> ", key, dnsAnswer.answer)
			if msg != nil {
				msg.Answer = dnsAnswer.answer
			}
			return
		}
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dnsType)
	m.RecursionDesired = true

	var c = new(dns.Client)

	c.ReadTimeout = time.Duration(1) * time.Second
	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		glog.Warningf("error: %s\n", err.Error())
		return
	}
	if r.Rcode != dns.RcodeSuccess {
		glog.Warningf(" invalid answer name %s after MX query for %s\n", domain, key)
		return
	}

	if msg != nil {
		msg.Answer = r.Answer
	}

	//glog.Infof(key+" -> Store answer to cache %s", r.Answer)
	dnsAnswerMap.Store(key, DNSAnswer{r.Answer, false, time.Now()})

	if wfJob != nil && r.Answer != nil && len(r.Answer) > 0 {
		wfJob <- &key
	}
}
