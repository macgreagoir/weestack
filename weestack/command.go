package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/macgreagoir/weestack/virtualmachines"
)

// createVMs, to call the build and install of virtual machines.
func createVMs() error {
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

	flagSet := flag.NewFlagSet("create", flag.ExitOnError)
	flagSet.StringVar(&bridge, "bridge", "virbr0", "Linux bridge these machines will use for networking")
	flagSet.StringVar(&domain, "domain", "example.com", "DNS domain name")
	ipAddrsString := flagSet.String("ip-addresses", "", "Comma-separated list of IP addresses, one per machine to be built")
	flagSet.StringVar(&netMask, "network-mask", "255.255.255.0", "Network mask of IP address")
	flagSet.StringVar(&gateway, "gateway", "", "Default network gateway for machines")
	flagSet.StringVar(&nameserver, "nameserver", "", "DNS nameserver for machines")
	flagSet.StringVar(&password, "password", "", "A cleartext password, which will be used for the debian and root users")
	flagSet.StringVar(&sshKeysUrl, "ssh-keys-url", "", "A URL for SSH key(s) to be used by the debian user")
	flagSet.Parse(os.Args[2:])

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
			flagSet.PrintDefaults()
			return errors.New("invalid options")
		}
	}
	if err := virtualmachines.ValidIPAddrs(ipAddrs); err != nil {
		return err
	}
	if err := virtualmachines.ValidNetMask(netMask); err != nil {
		return err
	}
	if err := virtualmachines.ValidGateway(gateway); err != nil {
		return err
	}
	if err := virtualmachines.ValidNameserver(nameserver); err != nil {
		return err
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
	if err := virtualmachines.Create(config); err != nil {
		return err
	}
	return nil
}

// deleteVMs, to call the removal of virtual machines.
func deleteVMs() error {
	var machines []string

	flagSet := flag.NewFlagSet("delete", flag.ExitOnError)
	machinesStr := flagSet.String("machines", "", "Comma-separated list of machines to remove, either IP addresses or machine names")
	flagSet.Parse(os.Args[2:])

	if *machinesStr == "" {
		flagSet.PrintDefaults()
		return errors.New("machines list is empty")
	}
	machines = strings.Split(*machinesStr, ",")
	if err := virtualmachines.Delete(machines); err != nil {
		return err
	}
	return nil
}
