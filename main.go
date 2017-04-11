package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/macgreagoir/weestack/hostmachine"
	"github.com/macgreagoir/weestack/virtualmachines"
)

var (
	bridge     string
	domain     string
	ipAddrs    []string
	netMask    string
	gateway    string
	nameserver string
	password   string
	sshKeysUrl string
)

func init() {
	flag.StringVar(&bridge, "bridge", "virbr0", "Linux bridge these machines will use for networking")
	flag.StringVar(&domain, "domain", "example.com", "DNS domain name")
	ipAddrsString := flag.String("ip-addresses", "", "Comma-separated list of IP addresses, one per machine to be built")
	flag.StringVar(&netMask, "network-mask", "255.255.255.0", "Network mask of IP address")
	flag.StringVar(&gateway, "gateway", "", "Default network gateway for machines")
	flag.StringVar(&nameserver, "nameserver", "", "DNS nameserver for machines")
	flag.StringVar(&password, "password", "", "A cleartext password, which will be used for the debian and root users")
	flag.StringVar(&sshKeysUrl, "ssh-keys-url", "", "A URL for SSH key(s) to be used by the debian user")
	flag.Parse()

	ipAddrs = strings.Split(*ipAddrsString, ",")
	for _, required := range []string{
		bridge,
		domain,
		netMask,
		gateway,
		nameserver,
		password,
		sshKeysUrl,
	} {
		if required == "" {
			fmt.Println("All command-line options must be set or left with their defaults:")
			flag.PrintDefaults()
			log.Fatal("invalid options")
		}
	}
	if err := virtualmachines.ValidIPAddrs(ipAddrs); err != nil {
		log.Fatal(err)
	}
	if err := virtualmachines.ValidNetMask(netMask); err != nil {
		log.Fatal(err)
	}
	if err := virtualmachines.ValidGateway(gateway); err != nil {
		log.Fatal(err)
	}
	if err := virtualmachines.ValidNameserver(nameserver); err != nil {
		log.Fatal(err)
	}
}

func main() {
	// TODO This is a wee bit lazy, using root to install pkgs and
	// configure directories.
	if err := hostmachine.Root(); err != nil {
		log.Fatal(err)
	}
	if err := hostmachine.InstallVirt(); err != nil {
		log.Fatal(err)
	}
	config := virtualmachines.Config{
		Bridge:     bridge,
		Domain:     domain,
		IPAddrs:    ipAddrs,
		NetMask:    netMask,
		Gateway:    gateway,
		Nameserver: nameserver,
		Password:   password,
		SSHKeysURL: sshKeysUrl,
	}
	virtualmachines.Create(config)
}
