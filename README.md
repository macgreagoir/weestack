WeeStack
========

A trivial 'local cloud' manager, when you don't need a big stack.

```
weestack create \
    --password s3cr3t \
    --ssh-keys-url https://example.com/me/sshkeys \
    --gateway 192.168.122.1 \
    --nameserver 192.168.122.1 \
    --ip-addresses 192.168.122.101,192.168.122.102  # one per VM
```

What It Does
------------

`sudo weestack init`

 * Installs the requirements

`weestack create`

 * Adds multiple KVM virtual machines on the local hypervisor
 * Installs them with Debian Jessie (8)
 * Configures them with a static IP address of your choice, on a bridge you may specify
   * You will likely want to manually update your libvirt networks to allow for a static range
 * Adds a 'debian' user with an authorized SSH key of your choice, and passwordless sudo

`weestack delete`

 * Removes machines, just like `virsh undefine --storage vda <name>`

What It Does Not Do (Yet)
-------------------------

 * Manage libvirt networks
 * Routable networking or floating IP addresses
 * IPv6
 * Manage containers or other non-KVM machines
 * Use prebuilt images
