Linux-kernel-fast-logging-path
==============================

Patch for linux kernel for building lists on changed files without any overhead

Author: Pavel Bolding, boldin.pavel [at] gmail.com

Debian Squeeze with Ubuntu kernel build manual:

```bash
apt-get update;
cd /usr/src

# Soft for .deb build
apt-get install -y dpkg-dev devscripts build-essential fakeroot

# Instal dependency, get it from debuid -us -uc output
apt-get install -y libelf-dev binutils-dev libdw-dev xmlto docbook-utils transfig asciidoc

# Get latest version from:
# http://packages.ubuntu.com/lucid/linux-image-2.6.35-32-server
# my mirror for this kernels: https://fastvps.googlecode.com/files/linux-lts-backport-maverick_2.6.35-32.68~lucid1.tar.gz and
# https://fastvps.googlecode.com/files/linux-lts-backport-maverick_2.6.35-32.68~lucid1.dsc.txt
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux-lts-backport-maverick/linux-lts-backport-maverick_2.6.35-32.68~lucid1.dsc
wget http://archive.ubuntu.com/ubuntu/pool/main/l/linux-lts-backport-maverick/linux-lts-backport-maverick_2.6.35-32.68~lucid1.tar.gz



# Apply patches
dpkg-source -x linux-lts-backport-maverick_2.6.35-32.68~lucid1.dsc

cd linux-lts-backport-maverick-2.6.35

# Remove makedumpfile dependency
#sed -i 's/makedumpfile \[amd64 i386\], //' debian/control

# Remove wireless-crda dependency
sed -i 's/, wireless-crda//g' debian/control
sed -i 's/, wireless-crda//g' debian/control.stub
sed -i 's/, wireless-crda//g' debian.master/control
sed -i 's/, wireless-crda//g' debian.master/control.stub
sed -i 's/, wireless-crda//g' debian.master/control.d/flavour-control.stub

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
# install new kernel (if u build it corectly, u don't need param ignore-depends)
dpkg --ignore-depends=wireless-crda -i linux-image-2.6.35-32-generic_2.6.35-32.68~lucid1_amd64.deb
```

Customization in booted system.

You need create miunt point for debugfs:
```bash
mkdir /mnt/debugfs
```

And add this line to /etc/fstab for enable mounting of debugfs filesystem:
```bash
none /mnt/debugfs debugfs
```

After that, you need reboot server or call mount -a command.

After that, you need enable backup logging in sysctl:
```bash
echo "kernel.fastvps_logging_user=1" >> /etc/sysctl.conf 
echo "kernel.fastvps_logging_root=1" >> /etc/sysctl.conf
sysctl -p 
```

After that, u will see log files by count of logical processors in the system (zero size is ok, don't worry):
```bash
ls -al /mnt/debugfs/backup/
total 0
drwxr-xr-x  2 root root 0 Aug 12 23:06 .
drwxr-xr-x 10 root root 0 Aug 12 23:06 ..
-r--------  1 root root 0 Aug 12 23:06 log0
-r--------  1 root root 0 Aug 12 23:06 log1
-r--------  1 root root 0 Aug 12 23:06 log2
-r--------  1 root root 0 Aug 12 23:06 log3
-r--------  1 root root 0 Aug 12 23:06 log4
-r--------  1 root root 0 Aug 12 23:06 log5
-r--------  1 root root 0 Aug 12 23:06 log6
-r--------  1 root root 0 Aug 12 23:06 log7
```

This files looks like dmesg, if u read files, file size goes to zero:
```bash
cat /mnt/debugfs/backup/log1
0 REM /var/lib/apt/lists/partial/mirror.hetzner.de_debian_packages_dists_squeeze_Release.gpg.reverify
0 NEW /var/lib/apt/lists/partial/mirror.hetzner.de_debian_packages_dists_squeeze_Release.gpg
0 REM /var/lib/apt/lists/partial/mirror.hetzner.de_debian_security_dists_squeeze_updates_Release.gpg.reverify
0 NEW /var/lib/apt/lists/partial/mirror.hetzner.de_debian_security_dists_squeeze_updates_Release.gpg
0 REM /var/lib/apt/lists/partial/mirror.hetzner.de_debian_backports_dists_squeeze-backports_Release.gpg.reverify
0 NEW /var/lib/apt/lists/partial/mirror.hetzner.de_debian_backports_dists_squeeze-backports_Release.gpg
```

And second call show no data:
```bash
cat /mnt/debugfs/backup/log1
# 
```

Also you need add this script  to cron (every 5 minutes) for prepare full list of changed files:

This script creates full incremental backup log:
```bash
cat /var/log/backup/2013-08-12.log 
0 MOD /root/4913
0 MOD /root/flush_backups_to_log.pl
0 MOD /root/.flush_backups_to_log.pl.swp
0 MOD /root/.viminfo.tmp
0 MOD /var/log/backup
0 NEW /root/4913
...
```

FAQ:
* What patch I need to use for kernel 2.6.33.5? You need use: fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v3_2_6_33_5.patch
*  What patch I need to use for kernel 2.6.35? You need use: fastvps-hosting-backup-with-chroot-and-var-backup-ignore-v4_2_6_35.patch
* What kernel guarantee stable work with this patch? We use v4 kernel with 3.6.35 few years without any issues.
* This patch is slow down kernel? Not, it's very light patch without performance killer features
* Do I need create custom backup script for this patch? Yes, I do not know any backup system for this patch.
* What mean MOD/NEW/REM in backup log? Modify, new and remove :)

Known issues: 
* If you have many-many-many files changed so recently, you may got errror: "FVLOG: Buffer for cpu 0 is about 80% full" in dmesg need reduce cron start interval for script flush_backups_to_log.pl
