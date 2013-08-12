#!/usr/bin/perl

use strict;
use warnings;

#
# With 15 minutes-flush we has:
# [1072451.030538] FVLOG: Buffer for cpu 0 is about 80% full
# [1072514.311969] FVLOG: Buffer overflow for cpu 0!
# [1073062.532398] FVLOG: Buffer for cpu 0 is about 80% full
#
# Up to 5 minutes :)
#

my $DEBUG = 0;

my $kernel_log_path = '/mnt/debugfs/backup';
my $log_path = '/var/log/backup';

my $chmod = '/bin/chmod';
my $mkdir = '/bin/mkdir';
my $date  = '/bin/date';
my $cat   = '/bin/cat';
my $grep  = '/bin/grep';

# Add path for ignoring from full list of files here
my @exclude_paths = (
    #'/data/bin-tmp/sess_',
    #'/var/log/',
    #'/data/logs/',  
    #'/var/backup/',
    #'/var/spool/postfix/',
    #'/var/lib/nginx/',           
    #'/var/lib/php5/',           
);

system "$mkdir -p  $log_path";
system "$chmod 700 $log_path";

# backup date format
my $backup_date = `$date +%Y-%m-%d`;
chomp $backup_date;

my $backup_log_path = "$log_path/$backup_date.log";

# build expclude string
@exclude_paths  = map { "grep -v '$_'" } @exclude_paths; 

my $exclude_list = join ' | ', @exclude_paths;

# count number of processes
my $processor_number = `$grep -c '^processor' /proc/cpuinfo`;
chomp $processor_number;

unless ($processor_number && $processor_number =~ /^\d+$/) {
    print "Can't get processor count!\n";
    exit 1;
}

for my $num (0 .. $processor_number - 1) {
   if (-e "$kernel_log_path/log$num") {
       my $log_file_name = "$kernel_log_path/log$num";

       if ($DEBUG) {
           print "parse: $log_file_name\n";
       }      


       # flush buffer to log with sessions exclude (it about 60% of all changed files)
       system "$cat $log_file_name | $exclude_list | sort | uniq  >> $backup_log_path" 
    }
}

