Linux-kernel-fast-logging-path
==============================

Path for linux kernel for building lists on changed files without any overhead


Ubuntu 2.6.35 build pathc

```bash
# #!/bin/sh
From https://progress.fastvps.ru/view.php?id=508

apt-get update;
cd /usr/src

# Soft for .deb build
apt-get install -y dpkg-dev devscripts build-essential fakeroot

# Instal dependency, get it from debuid -us -uc output
apt-get install -y libelf-dev binutils-dev libdw-dev xmlto docbook-utils transfig asciidoc

# Get latest version from:
# http://packages.ubuntu.com/maverick/linux-source-2.6.35
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux/linux_2.6.35-23.41.dsc
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux/linux_2.6.35.orig.tar.gz
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux/linux_2.6.35-23.41.diff.gz

# Apply patches
dpkg-source -x linux_2.6.35-23.41.dsc

cd linux-2.6.35

# Remove makedumpfile dependency
sed -i 's/makedumpfile \[amd64 i386\], //' debian/control

# Remove wireless-crda dependency
sed -i 's/, wireless-crda//g' debian/control
sed -i 's/, wireless-crda//g' debian/control.stub
sed -i 's/, wireless-crda//g' debian.master/control
sed -i 's/, wireless-crda//g' debian.master/control.stub
sed -i 's/, wireless-crda//g' debian.master/control.d/flavour-control.stub

# Null flavour configs
cp /dev/null debian.master/config/amd64/config.flavour.generic
cp /dev/null debian.master/config/amd64/config.flavour.server 
cp /dev/null debian.master/config/amd64/config.flavour.virtual 

# Clean up ubuntu kernel requirements
cp /dev/null debian.master/config/enforce 

# Get our config
wget http://download.fastvps.ru/dist/kernel_configs/config-2.6.35-23-generic -Odebian.master/config/amd64/config.common.amd64

# Bluehost backup patch
wget http://download.fastvps.ru/dist/fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch
patch -p1 < fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch

# Build kernel
no_dumpfile=true skipabi=true skipmodule=true debian/rules binary-generic

cd ..
```
