package virtualmachines

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"text/template"
)

const (
	// TODO This is a wee bit open, but libvirt seems to need it :-/
	ModeRW  os.FileMode = 0664
	ModeRWX os.FileMode = 0750
)

var (
	LocalDir = filepath.Join(os.Getenv("HOME"), ".local", "share", "weestack")
	// TODO What's the point of using Join here if it doesn't know an
	// OS-independent 'root' dir?
	LibvirtDir = filepath.Join("/", "var", "lib", "libvirt")
)

var preseedCfg = `
d-i debian-installer/locale select en_GB.UTF-8
d-i keyboard-configuration/xkb-keymap select gb
d-i netcfg/choose_interface select auto
d-i netcfg/disable_autoconfig boolean true
d-i netcfg/get_ipaddress string {{.IPAddr}}
d-i netcfg/get_netmask string {{.NetMask}}
d-i netcfg/get_gateway string {{.Gateway}}
d-i netcfg/get_nameservers string {{.Nameserver}}
d-i netcfg/confirm_static boolean true
d-i netcfg/get_hostname string {{.Hostname}}
d-i netcfg/get_domain string {{.Domain}}
d-i netcfg/wireless_wep string
d-i mirror/country string IE
d-i mirror/http/mirror select ftp.ie.debian.org
d-i mirror/http/hostname string ftp.ie.debian.org
d-i mirror/http/directory string /debian/
d-i mirror/http/proxy string
d-i passwd/root-password password {{.Password}}
d-i passwd/root-password-again password {{.Password}}
d-i passwd/user-fullname string Debian
d-i passwd/username string debian
d-i passwd/user-password password {{.Password}}
d-i passwd/user-password-again password {{.Password}}
d-i clock-setup/utc boolean true
d-i time/zone string Europe/Dublin
d-i clock-setup/ntp boolean true
d-i partman-auto/method string regular
d-i partman-lvm/device_remove_lvm boolean true
d-i partman-md/device_remove_md boolean true
d-i partman-lvm/confirm boolean true
d-i partman-lvm/confirm_nooverwrite boolean true
d-i partman-auto/choose_recipe select atomic
d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true
d-i partman-md/confirm boolean true
d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true
tasksel tasksel/first multiselect standard, ssh-server
d-i pkgsel/include string curl sudo vim wget
popularity-contest popularity-contest/participate boolean false
d-i grub-installer/only_debian boolean true
d-i grub-installer/with_other_os boolean true
d-i grub-installer/bootdev  string /dev/vda
d-i debian-installer/add-kernel-opts string \
  console=tty0 console=ttyS0,115200 console=ttyS1,115200 panic=30 raid=noautodetect
d-i finish-install/reboot_in_progress note
d-i debian-installer/exit/poweroff boolean true
d-i preseed/late_command string echo "DOTSSH=/home/debian/.ssh; mkdir \$DOTSSH; wget -O \$DOTSSH/authorized_keys {{.SSHKeysURL}}; chmod 700 \$DOTSSH; chmod 400 \$DOTSSH/authorized_keys; chown -R debian:debian \$DOTSSH; SUDOERSD=/etc/sudoers.d/debian; echo 'debian ALL=(ALL) NOPASSWD: ALL' >> \$SUDOERSD; echo 'Defaults:debian !requiretty' >> \$SUDOERSD; chmod 0440 \$SUDOERSD" | chroot /target /bin/bash;
`[1:]

// PreseedsConfig holds the configuraton of the generic preseed used to build
// individual machine preseeds.
type PreseedsConfig struct {
	// dir is path to the parent preseeds directory, inside which per
	// machine directories will store preseed.cfg.
	dir string

	// tmpl is the template from which preseed.cfg files will be built.
	tmpl *template.Template
}

// Dir gets and/or sets the preseeds parent directory. If the call to mkdir
// fails, the dir field will be left with its zero value.
func (p *PreseedsConfig) Dir() (string, error) {
	if p.dir != "" {
		return p.dir, nil
	}
	dir := filepath.Join(LocalDir, "preseeds")
	if err := os.MkdirAll(dir, ModeRWX); err != nil {
		return "", err
	}
	p.dir = dir
	return p.dir, nil
}

// Path returns a path with name as base to the preseeds dir.
func (p *PreseedsConfig) Path(name string) (string, error) {
	dir, err := p.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), err
}

// Tmpl gets and/or sets the template used to write out preseeds.
func (p *PreseedsConfig) Tmpl() (*template.Template, error) {
	if p.tmpl != nil {
		return p.tmpl, nil
	}
	t, err := template.New("preseed.tmpl").Parse(preseedCfg)
	if err != nil {
		return nil, err
	}
	p.tmpl = t
	return p.tmpl, nil
}

var Preseeds PreseedsConfig

// DisksConfig holds the configuraton of the generic disk configuration used to
// build individual machine disks.
type DisksConfig struct {
	// dir is path to the disk images.
	dir string

	// size of disk, for example "10G".
	size string

	// format is likely "qcow2".
	format string

	// ext is the image file suffix, and is
	// likely "qcow2".
	ext string
}

// Dir gets and/or sets the Disks parent directory. If the call to mkdir fails,
// the dir field will be left with its zero value.
func (d *DisksConfig) Dir() (string, error) {
	if d.dir != "" {
		return d.dir, nil
	}
	dir := filepath.Join(LibvirtDir, "weestack")
	if err := os.MkdirAll(dir, ModeRWX); err != nil {
		return "", err
	}
	if err := Chown(dir, "libvirt-qemu"); err != nil {
		return "", err
	}
	d.dir = dir
	return d.dir, nil
}

// Path returns a path with name as base to the disks dir.
func (d *DisksConfig) Path(name string) (string, error) {
	dir, err := d.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), err
}

var Disks DisksConfig

// TODO For now, just effectively set some constants. These should probably be
// config options.
func init() {
	Disks.size = "10G"
	Disks.format = "qcow2"
	Disks.ext = "qcow2"
}

// Exists tests that a file exists.
// TODO This probably belongs in some more general utils package.
func Exists(name string) (bool, error) {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// We can't say the file does not exist because we got some
		// other error. Err of the side of caution.
		return true, err
	}
	return true, nil
}

// Chown uses os.Chown, but allows a username string instead or requiring
// knowledge of uid and gid.
func Chown(filename string, username string) error {
	u, err := user.Lookup(username)
	if err != nil {
		return err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	err = os.Chown(filename, uid, gid)
	if err != nil {
		return err
	}
	return nil
}
