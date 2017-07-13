package domwatch

import (
	"errors"
	"fmt"

	"log"

	"strings"

	"github.com/miekg/dns"
)

func IsDomainAvailable(server string, domain string, transport string, types []uint16, debugLogger *log.Logger) (bool, error) {
	var err error
	var nameServers []string
	nameServers, err = getNameServers(server, domain, transport, debugLogger)
	if err != nil {
		return false, err
	}
	if nameServers == nil || len(nameServers) <= 0 {
		return false, fmt.Errorf("Unable to find nameservers for '%s'", domain)
	}

	domain = domain + "."

	var client dns.Client
	var request dns.Msg
	var response *dns.Msg
	client.Net = transport

	domainExists := false

	for _, ns := range nameServers {
		debugLogger.Printf("Querying '%s'\n", ns)
		request.Question = make([]dns.Question, len(types))
		for i := len(types) - 1; i >= 0; i-- {
			request.Question[i] = dns.Question{
				Name:   domain,
				Qtype:  types[i],
				Qclass: dns.ClassINET,
			}
		}
		response, _, err = client.Exchange(&request, ns+":53")
		if err != nil {
			debugLogger.Printf("Error from nameserver %s: %s", ns, err.Error())
			continue
		}

		if len(response.Answer) > 0 {
			domainExists = true
			break
		}
	}

	if domainExists {
		debugLogger.Printf("%s is not available", domain)
	} else {
		debugLogger.Printf("%s is available", domain)
	}
	return !domainExists, nil
}

func getNameServers(server string, domain string, transport string, debugLogger *log.Logger) ([]string, error) {
	var err error
	var client dns.Client
	var request dns.Msg
	var response *dns.Msg
	client.Net = transport

	debugLogger.Printf("Getting root ns for '%s'\n", domain)

	domainParts := strings.Split(domain, ".")
	if len(domainParts) <= 1 {
		return nil, errors.New("Invalid domain")
	}
	tld := domainParts[len(domainParts)-1] + "."
	request.SetQuestion(tld, dns.TypeNS)
	response, _, err = client.Exchange(&request, server+":53")

	if err != nil {
		return nil, err
	}

	l := len(response.Answer)

	if l <= 0 {
		return nil, fmt.Errorf("No nameservers found for '%s'", tld)
	}

	var servers []string

	for i := 0; i < l; i++ {
		switch response.Answer[i].(type) {
		case *dns.NS:
			servers = append(servers, strings.TrimSpace(strings.Trim(response.Answer[i].(*dns.NS).Ns, ".")))
		}
	}

	return servers, nil
}
