Linux-kernel-fast-logging-path
==============================

Path for linux kernel for building lists on changed files without any overhead

Author: Pavel Bolding, boldin.pavel [at] gmail.com

Ubuntu 2.6.35 build pathc

```bash
# #!/bin/sh

apt-get update;
cd /usr/src

# Soft for .deb build
apt-get install -y dpkg-dev devscripts build-essential fakeroot

# Instal dependency, get it from debuid -us -uc output
apt-get install -y libelf-dev binutils-dev libdw-dev xmlto docbook-utils transfig asciidoc

# Get latest version from:
# http://packages.ubuntu.com/lucid/linux-image-2.6.35-32-server
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux-lts-backport-maverick/linux-lts-backport-maverick_2.6.35-32.68~lucid1.dsc
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux-lts-backport-maverick/linux-lts-backport-maverick_2.6.35-32.68~lucid1.tar.gz



# Apply patches
dpkg-source -x linux-lts-backport-maverick_2.6.35-32.68~lucid1.dsc

cd linux-lts-backport-maverick-2.6.35

# Remove makedumpfile dependency
#sed -i 's/makedumpfile \[amd64 i386\], //' debian/control

# Remove wireless-crda dependency
#sed -i 's/, wireless-crda//g' debian/control
#sed -i 's/, wireless-crda//g' debian/control.stub
#sed -i 's/, wireless-crda//g' debian.master/control
#sed -i 's/, wireless-crda//g' debian.master/control.stub
#sed -i 's/, wireless-crda//g' debian.master/control.d/flavour-control.stub

# Clean up ubuntu kernel requirements
### cp /dev/null debian.master/config/enforce 

#### Get our config
### wget http://..../kernel_configs/config-2.6.35-23-generic -Odebian.master/config/amd64/config.common.amd64

# FastVPS backup patch
wget https://raw.github.com/FastVPSEestiOu/Linux-kernel-fast-logging-path/master/fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch
patch -p1 < fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch

# Build kernel
no_dumpfile=true skipabi=true skipmodule=true debian/rules binary-generic

cd ..
```

Customization in booted system.

You need add this line to /etc/fstab for enable mounting of debugfs filesystem:
```bash
none /mnt/debugfs debugfs
```

After that, you need reboot server or call mount -a command.

Also you need add this script  to cron (every 5 minutes) for prepare full list of changed files: https://raw.github.com/FastVPSEestiOu/Linux-kernel-fast-logging-path/master/flush_backups_to_log.pl

FAQ:
* What patch I need to use for kernel 2.6.33.5? You need use: fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v3_2_6_33_5.patch
*  What patch I need to use for kernel 2.6.35? You need use: fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch
* What kernel guarantee stable work with this patch? We use v4 kernel with 3.6.35 few years without any issues.
* This patch is slow down kernel? Not, it's very light patch without performance killer features
* Do I need create custom backup script for this patch? Yes, I do not know any backup system for this patch.

Known issues: 
* If you have many-many-many files changed so recently, you may got errror: "FVLOG: Buffer for cpu 0 is about 80% full" in dmesg need reduce cron start interval for script flush_backups_to_log.pl
