# Gox

Painless backup (and restore) tool for MySQL based on xtrabackup from Percona.

<div style="width:100%;height:0;padding-bottom:64%;position:relative;"><iframe src="https://giphy.com/embed/zcCGBRQshGdt6" width="100%" height="100%" style="position:absolute" frameBorder="0" class="giphy-embed" allowFullScreen></iframe></div>

## Quick start


### Install xtrabackup and qpress on MySQL server

See [install procedure here](https://www.percona.com/doc/percona-xtrabackup/2.4/index.html#installation) for xtrabackup

Once you have installed xtrabackup, install qpress via:

```
apt-get install qpress
```

### Download gox on our backup server


### Create a config file corresponding to the backup/remote task

See [config.yalm.sample](config.yalm.sample)

Here is a sample:

```yaml
# Remote MySQL host
host: mysql.explample.com
# Mysql user
dbuser: root
# Mysql password
dbpassword:
# SSH config
ssh:
  # SSH user
  user: root
  # Private key for ssh user
  key: /home/jdoe/.ssh/id_rsa
# Remote path of xtrabsckup binary
xtrabackup: /usr/bin/xtrabackup
# The number of threads to use to copy multiple data files concurrently when creating a backup
parallel: 2
# Compression 
compress:
  # Compress ?
  active: true
  # This option specifies the number of worker threads used by xtrabackup for parallel data compression
  threads: 2
  # This option when specified will remove .qp, .xbcrypt and .qp.xbcrypt files after decryption and decompression.
  remove-original: true
# This options creates the xtrabackup_galera_info file which contains the local node state at the time of the backup. 
galera: false
# Storage path for backup
backup-dir: /tmp/
# Remove backups older than 'keep' 
# A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix,
# such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
keep: 168h
```

### Backup

```
gox backup --config /path/to/mysql.host.com.yaml
```

### Restore

```
gox restore --config /path/to/mysql.host.com.yaml --from 2018-01-31--08-32-49
```
