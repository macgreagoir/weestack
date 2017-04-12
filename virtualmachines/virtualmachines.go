// virtualmachines manages the WeeStack virtual machines.
package virtualmachines

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config holds the values of passed-in flags as the WeeStack configuration.
type Config struct {
	// Bridge is the Linux bridge all machines use for networking.
	Bridge string

	// Domain is the DNS domain name used by all machines.
	Domain string

	// IPAddrs is the list of host IP addresses, one per machine.
	IPAddrs []string

	// NetMask is the IP address network mask.
	NetMask string

	// Gateway is the default gateway IP address.
	Gateway string

	// Nameserver is the DNS nameserver IP address.
	// TODO This should really be list of addresses.
	Nameserver string

	// Password is the cleartext password used for both the root and debian
	// users on each machine.
	// TODO This is not really required and could be optional.
	Password string

	// SSHKeysURL is the location of the SSH keys to be used as
	// authorized_keys for the debian user.
	SSHKeysURL string
}

// machine holds the configuration of a single virtual machine.
type machine struct {
	Config

	// IPAddr is this machine's IP addr.
	IPAddr string

	// Hostname is this machine's hostname.
	Hostname string

	// preseed is the absolute path of this machine's preseed.cfg.
	preseed string
}

// ValidIPAddrs checks that ipAddrs is a list of valid IP addresses.
func ValidIPAddrs(ipAddrs []string) error {
	for _, ipAddr := range ipAddrs {
		if ipAddr == "" {
			fmt.Printf(`
IP addresses must be passed in as a comma-separated list, one per machine to be built.
For example, '--ip-addresses 192.168.122.101,192.168.122.102'
`[1:])
			return errors.New("IP address is empty")
		}
		if err := validIPAddr(ipAddr); err != nil {
			return err
		}
	}
	return nil
}

// ValidNetMask checks that netMask looks like a valid IP address.
func ValidNetMask(netMask string) error {
	if err := validIPAddr(netMask); err != nil {
		return err
	}
	return nil

}

// ValidGateway checks that gateway looks like an IP address.
func ValidGateway(gateway string) error {
	return validIPAddr(gateway)
}

// ValidNameserver checks that nameserver looks like an IP address.
func ValidNameserver(nameserver string) error {
	return validIPAddr(nameserver)
}

// validIPAddr checks that ipAddr looks like an IP address.
func validIPAddr(ipAddr string) error {
	if addr := net.ParseIP(ipAddr); addr == nil {
		return errors.New(
			fmt.Sprintf("%s is not a valid IP address\n", ipAddr),
		)
	}
	return nil
}

// Create creates the virtual machines from the config.
func Create(config Config) error {
	c := make(chan error, len(config.IPAddrs))
	for _, ipAddr := range config.IPAddrs {
		m := machine{
			Config: Config{
				Bridge:     config.Bridge,
				Domain:     config.Domain,
				NetMask:    config.NetMask,
				Gateway:    config.Gateway,
				Nameserver: config.Nameserver,
				Password:   config.Password,
				SSHKeysURL: config.SSHKeysURL,
			},
			IPAddr: ipAddr,
			// TODO IPv4 requirement, also seen in preseedCfg.
			Hostname: strings.Replace(ipAddr, ".", "-", -1),
		}
		go func() {
			if err := m.createPreseed(); err != nil {
				c <- err
				return
			}
			c <- m.createMachine()
		}()
	}
	return errChan("creating machines", c)
}

// createPreseed writes out the Debian preseed.cfg file for the machine.
func (m *machine) createPreseed() error {
	d, err := Preseeds.Path(m.Hostname)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, ModeRWX); err != nil {
		return err
	}
	f, err := os.OpenFile(
		filepath.Join(d, "preseed.cfg"),
		os.O_RDWR|os.O_CREATE,
		ModeRW,
	)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Printf("writing %s\n", f.Name())
	pTmpl, err := Preseeds.Tmpl()
	if err != nil {
		return err
	}
	if err := pTmpl.Execute(f, m); err != nil {
		return err
	}
	m.preseed = f.Name()
	return nil
}

// createMachine builds a single virtual machine.
func (m *machine) createMachine() error {
	// TODO Here we shell out to use `qemu-img` and `virt-install`. It
	// would be nice to use something like `libvirt-go`, but `virt-install`
	// gives useful features like preseed injection, and qemu might need a
	// whole new Golang binding package.

	disk, err := Disks.Path(fmt.Sprintf("%s.%s", m.Hostname, Disks.ext))
	if err != nil {
		return err
	}

	exists, err := Exists(disk)
	if err != nil {
		return err
	}
	if !exists {
		cmd := exec.Command(
			"/usr/bin/qemu-img", "create",
			"-f", Disks.format,
			disk,
			Disks.size,
		)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return errors.New(stderr.String())
		}
	}
	if err := Chown(disk, "libvirt-qemu"); err != nil {
		return err
	}

	cmd := exec.Command(
		"/usr/bin/virt-install",
		"--connect", "qemu:///system",
		"--virt-type", "kvm",
		"--name", m.Hostname,
		"--cpu", "host-model-only",
		"--vcpus", "2",
		"--ram", "2048",
		"--disk", disk,
		"--location", "http://ftp.debian.org/debian/dists/jessie/main/installer-amd64/",
		"--initrd-inject", m.preseed,
		"--extra-args", `"console=tty0 console=ttyS0,115200 console=ttyS1,115200 panic=30 raid=noautodetect"`,
		"--network", "bridge="+m.Bridge,
		"--graphics", "none",
		"--os-type", "linux",
		"--os-variant", "debian8",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}
	log.Printf("creating machine %q\n", m.Hostname)
	return nil
}

// Delete removes virtual machines and their storage.
// It takes a slice of machines, which can be either IP addresses, to
// be consistent with Create, or VM names.
func Delete(machines []string) error {
	c := make(chan error, len(machines))
	for _, machine := range machines {
		// TODO Is there any reason to check this looks like an
		// IP addr first and only replace on ones that do?
		name := strings.Replace(machine, ".", "-", -1)
		go func() {
			c <- deleteMachine(name)
		}()
	}
	return errChan("deleting machines", c)
}

// deleteMachine deletes individual virtual machines.
// TODO Use libvirt-go instead of shelling out.
func deleteMachine(name string) error {
	cmd := exec.Command(
		"/usr/bin/virsh", "undefine", name,
		"--storage", "vda",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.New(stderr.String())
	}
	log.Printf("deleted machine %q\n", name)
	return nil
}

// errChan returns errors received on an errors channel as a single
// error, or nil if no errors were received.
// desc is a short prefix to describe the errors' context in the new
// error, for example, "creating machine".
func errChan(desc string, c chan error) error {
	var errs []string
	// TODO There is an assumption here that the buffer is full.
	for i := 0; i < cap(c); i++ {
		err := <-c
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return errors.New(fmt.Sprintf(
			"Errors %s:\n%s",
			desc,
			strings.Join(errs, ""),
		))
	}
	return nil
}
