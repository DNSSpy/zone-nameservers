// Go equivalent of the "DNS & BIND" book check-soa program.
// Created by Stephane Bortzmeyer.
package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/miekg/dns"
)

const (
	// DefaultTimeout is default timeout many operation in this program will
	// use.
	DefaultTimeout time.Duration = 5 * time.Second
)

func init() {
	rand.Seed(time.Now().Unix())
}

type ZoneNsResolver struct {
	localm *dns.Msg
	localc *dns.Client
}

func NewZoneNsResolver() *ZoneNsResolver {
	return &ZoneNsResolver{
		&dns.Msg{
			MsgHdr: dns.MsgHdr{
				RecursionDesired: true,
			},
			Question: make([]dns.Question, 1),
		},
		&dns.Client{
			ReadTimeout: DefaultTimeout,
		},
	}
}

func (zr *ZoneNsResolver) localQuery(qname string, qtype uint16, server string) (*dns.Msg, error) {
	zr.localm.SetQuestion(qname, qtype)

	r, _, err := zr.localc.Exchange(zr.localm, server+":53")
	if err != nil {
		return nil, err
	}
	if r == nil || r.Rcode == dns.RcodeNameError || r.Rcode == dns.RcodeSuccess {
		return r, nil
	}

	return nil, errors.New("No name server to answer the question")
}

func (zr *ZoneNsResolver) Resolve(zone string, server string) ([]string, error) {
	zone = dns.Fqdn(zone)

	r, err := zr.localQuery(zone, dns.TypeNS, server)
	if err != nil || r == nil {
		return nil, err
	}

	var nameservers []string

	for _, ans := range r.Answer {
		if t, ok := ans.(*dns.NS); ok {
			nameserver := t.Ns
			nameservers = append(nameservers, nameserver)
		}
	}

	if len(nameservers) == 0 {
		// No "Answer" given by the server, check the Authority section if
		// additional nameservers were provided.
		for _, ans := range r.Ns {
			if t, ok := ans.(*dns.NS); ok {
				nameserver := t.Ns
				nameservers = append(nameservers, nameserver)
			}
		}
	}

	if len(nameservers) == 0 {
		return nil, errors.New("No nameservers found for " + zone)
	}

	sort.Strings(nameservers)

	return nameservers, nil
}

func domainToZones(domain string) []string {
	zones := []string{"."}

	assembled := ""
	pieces := dns.SplitDomainName(domain)
	for i := len(pieces) - 1; i >= 0; i-- {
		assembled = pieces[i] + "." + assembled
		zones = append(zones, assembled)
	}
	return zones
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("%s ZONE\n", os.Args[0])
	}
	domain := dns.Fqdn(os.Args[1])

	conf, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || conf == nil {
		log.Fatalf("Cannot initialize the local resolver: %s\n", err)
	}

	resolver := NewZoneNsResolver()

	var ns []string
	nextNs := conf.Servers[0]

	// split domain, query each part for NS records
	for _, zone := range domainToZones(domain) {
		if zone == "." {
			fmt.Println("Retrieving list of root nameservers:")
		} else {
			fmt.Println("\nFinding nameservers for zone '" + zone + "' using parent nameserver '" + nextNs + "'")
		}

		ns, err = resolver.Resolve(zone, nextNs)
		if err != nil {
			log.Fatalln("Query failed: ", err)
		}

		// Pick a random NS record for the next queries
		nextNs = ns[rand.Intn(len(ns))]

		// Print the nameservers for this zone, highlight the one we used to query
		for _, nameserver := range ns {
			if nameserver == nextNs && domain != zone {
				// We'll use this one for queries
				fmt.Println(" ➡️ " + nameserver)
			} else {
				fmt.Println(" - " + nameserver)
			}
		}
	}
}
