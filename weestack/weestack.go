package main

import (
	"fmt"
	"log"
	"os"

	"github.com/macgreagoir/weestack/hostmachine"
)

func usage() {
	helpText := `
WeeStack is a 'local cloud' manager, for when you don't need a big stack.
It will create and delete virtual machines on your local host.

Examples:

  sudo weestack init

  weestack create \
    --password s3cr3t \
    --ssh-keys-url https://example.com/me/sshkeys \
    --gateway 192.168.122.1 \
    --nameserver 192.168.122.1 \
    --ip-addresses 192.168.122.101,192.168.122.102  # one per VM
`

	fmt.Println(helpText)
	fmt.Printf("Usage: %s [init|create|delete] ...\n", os.Args[0])
	os.Exit(0)
}

func main() {
	if len(os.Args) == 1 {
		usage()
	}
	switch os.Args[1] {
	case "init":
		if err := hostmachine.Root(); err != nil {
			log.Fatal(err)
		}
		if err := hostmachine.InstallVirt(); err != nil {
			log.Fatal(err)
		}
	case "create":
		if err := createVMs(); err != nil {
			log.Fatal(err)
		}
	case "delete":
		if err := deleteVMs(); err != nil {
			log.Fatal(err)
		}
	default:
		usage()
	}
}
