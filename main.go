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

var (
	localm           *dns.Msg
	localc           *dns.Client
	conf             *dns.ClientConfig
	domain           string
	assembledDomain string
	nextNs          string
)

func localQuery(qname string, qtype uint16, server string) (*dns.Msg, error) {
	localm.SetQuestion(qname, qtype)
	r, _, err := localc.Exchange(localm, server+":53")
	if err != nil {
		return nil, err
	}
	if r == nil || r.Rcode == dns.RcodeNameError || r.Rcode == dns.RcodeSuccess {
		return r, err
	}

	return nil, errors.New("No name server to answer the question")
}

func getNsRecords(zone string, server string) ([]string, string, error) {
	zone = dns.Fqdn(zone)
	r, err := localQuery(zone, dns.TypeNS, server)
	if err != nil || r == nil {
		log.Fatal("Cannot retrieve the list of name servers for %s: %s\n", zone, err)
	}

	var nameservers []string
	var random_ns string

	for _, ans := range r.Answer {
		switch t := ans.(type) {
		case *dns.NS:
			nameserver := t.Ns
			nameservers = append(nameservers, nameserver)
		}
	}

	if len(nameservers) == 0 {
		// No "Answer" given by the server, check the Authority section if
		// additional nameservers were provided.
		for _, ans := range r.Ns {
			switch t := ans.(type) {
			case *dns.NS:
				nameserver := t.Ns
				nameservers = append(nameservers, nameserver)
			}
		}
	}

	if len(nameservers) == 0 {
		return nil, "", errors.New("No nameservers found for " + zone)
	}

	// Pick a random NS record for the next queries
	random_ns = nameservers[rand.Intn(len(nameservers))]

	sort.Strings(nameservers)

	return nameservers, random_ns, nil

}

func main() {
	if len(os.Args) != 2 {
		log.Fatal("%s ZONE\n", os.Args[0])
	}
	domain = os.Args[1]

	rand.Seed(time.Now().Unix())
	var err error
	conf, err = dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil || conf == nil {
		log.Fatal("Cannot initialize the local resolver: %s\n", err)
	}
	localm = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			RecursionDesired: true,
		},
		Question: make([]dns.Question, 1),
	}
	localc = &dns.Client{
		ReadTimeout: DefaultTimeout,
	}

	// Walk the root until you find the authoritative nameservers
	fmt.Printf("Retrieving list of root nameservers:\n")
	root_nameservers, nextNs, err := getNsRecords(".", conf.Servers[0])
	if err != nil {
		log.Fatal("Query failed: ", err)
	}
	for _, nameserver := range root_nameservers {
		if nameserver == nextNs {
			// We'll use this one for queries
			fmt.Println(" ➡️ " + nameserver)
		} else {
			fmt.Println(" - " + nameserver)
		}
	}

	// We have list of root nameservers: split domain, query each part for NS records
	domain_pieces := dns.SplitDomainName(domain)
	assembledDomain = "."
	var ns []string

	for i := len(domain_pieces) - 1; i >= 0; i-- {
		fmt.Println("\n")
		element := domain_pieces[i]
		if assembledDomain == "." {
			assembledDomain = element + "."
		} else {
			assembledDomain = element + "." + assembledDomain
		}

		fmt.Println("Finding nameservers for zone '" + assembledDomain + "' using parent nameserver '" + nextNs + "'")
		ns, nextNs, err = getNsRecords(assembledDomain, nextNs)
		if err != nil {
			fmt.Println("Query failed: ", err)
		}

		// Print the nameservers for this zone, highlight the one we used to query
		for _, nameserver := range ns {
			if nameserver == nextNs && dns.Fqdn(domain) != assembledDomain {
				// We'll use this one for queries
				fmt.Println(" ➡️ " + nameserver)
			} else {
				fmt.Println(" - " + nameserver)
			}
		}
	}

}
