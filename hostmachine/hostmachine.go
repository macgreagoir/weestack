// Host configures the machine to host the WeeStack cloud.
package hostmachine

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"os/user"
)

func Root() error {
	current, err := user.Current()
	if err != nil {
		return err
	}
	if current.Uid != "0" {
		return errors.New("this must be run with root user privileges")
	}
	return nil
}

func InstallVirt() error {
	// NOTE Only APT for now
	cmd := exec.Command(
		"/usr/bin/apt-get", "-y", "install",
		"qemu-kvm",
		"libvirt-bin",
		"virtinst",
		"virt-manager",
		"virt-viewer",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Print(stdout.String())
		return errors.New(stderr.String())
	}
	return nil
}
