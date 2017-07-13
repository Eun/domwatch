package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"log"

	"strings"

	"github.com/Eun/domwatch"
	"github.com/asaskevich/govalidator"
	"github.com/miekg/dns"
)

func main() {
	flag.Bool("tcp", true, "")
	useUDP := flag.Bool("udp", false, "")
	useA := flag.Bool("a", false, "")
	useNS := flag.Bool("ns", true, "")
	useCNAME := flag.Bool("cname", false, "")
	useSOA := flag.Bool("soa", true, "")
	usePTR := flag.Bool("ptr", false, "")
	useMX := flag.Bool("mx", false, "")
	useTXT := flag.Bool("txt", false, "")
	useAAAA := flag.Bool("aaaa", false, "")
	useLOC := flag.Bool("loc", false, "")
	useSRV := flag.Bool("srv", false, "")
	useSPF := flag.Bool("spf", false, "")
	verbose := flag.Bool("verbose", false, "")

	flag.Parse()

	args := flag.Args()
	if len(args) <= 0 {
		fmt.Printf("usage: %s <options> host\n", path.Base(os.Args[0]))
		fmt.Println("    Options:")
		fmt.Println("    -tcp          Force TCP")
		fmt.Println("    -udp          Force UDP")
		fmt.Println("    -a            Use A as lookup")
		fmt.Println("    -aaaa         Use AAAA as lookup")
		fmt.Println("    -cname        Use CNAME as lookup")
		fmt.Println("    -loc          Use LOC as lookup")
		fmt.Println("    -mx           Use MX as lookup")
		fmt.Println("    -ns           Use NS as lookup")
		fmt.Println("    -ptr          Use PTR as lookup")
		fmt.Println("    -soa          Use SOA as lookup")
		fmt.Println("    -spf          Use SPF as lookup")
		fmt.Println("    -srv          Use SRV as lookup")
		fmt.Println("    -txt          Use TXT as lookup")
		fmt.Println("    -verbose      Verbose output")
		os.Exit(1)
	}

	transport := "tcp"
	if *useUDP == true {
		transport = "udp"
	}

	host := args[0]

	host = strings.TrimSpace(host)
	host = strings.Trim(host, ".")

	if !govalidator.IsDNSName(host) {
		fmt.Fprintf(os.Stderr, "'%s' is not a domain name\n", host)
		os.Exit(1)
	}

	if strings.Index(host, ".") <= 0 {
		fmt.Fprintf(os.Stderr, "'%s' is not a domain name\n", host)
		os.Exit(1)
	}

	var types []uint16

	if *useA == true {
		types = append(types, dns.TypeA)
	}
	if *useNS == true {
		types = append(types, dns.TypeNS)
	}
	if *useCNAME == true {
		types = append(types, dns.TypeCNAME)
	}
	if *useSOA == true {
		types = append(types, dns.TypeSOA)
	}
	if *usePTR == true {
		types = append(types, dns.TypePTR)
	}
	if *useMX == true {
		types = append(types, dns.TypeMX)
	}
	if *useTXT == true {
		types = append(types, dns.TypeTXT)
	}
	if *useAAAA == true {
		types = append(types, dns.TypeAAAA)
	}
	if *useLOC == true {
		types = append(types, dns.TypeLOC)
	}
	if *useSRV == true {
		types = append(types, dns.TypeSRV)
	}
	if *useSPF == true {
		types = append(types, dns.TypeSPF)
	}

	if len(types) == 0 {
		fmt.Fprintln(os.Stderr, "No type to query selected")
		os.Exit(1)
	}

	debugLogger := log.New(&devNullWriter{}, "", log.LstdFlags)
	if *verbose == true {
		debugLogger.SetOutput(os.Stderr)
	}

	available, err := domwatch.IsDomainAvailable("8.8.8.8", host, transport, types, debugLogger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if available {
		fmt.Printf("'%s' is AVAILABLE\n", host)
	} else {
		fmt.Printf("'%s' is NOT available\n", host)
	}
}

type devNullWriter struct {
}

func (*devNullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
