## Container filesystem performance tests

This directory contains the Vagrantfile, install script and python test files that test the container start/stop/remove performance on various filesystem.

- aufs
- devicemapper+loopback
- devicemapper+direct ()
- overlayfs
- btrfs
- zfs

The tests/start_stop_apaches.py stops N containers, which runs apache2, then stop and remove them. (default N = 20)

To run the tests on your mac, just do:

```
vagrant up [vm-name]
```
list of vm-name:
- devaufs
- devdmloop
- devdmdirect
- devoverlay
- devbtrfs
- devzfs

The tests result is in results directory. To run manual test (verbose mode and start 100 containers) after the vm is up:
```
vagrant ssh vm-name
cd /vagrant/tests
sudo python start_stop_apaches.py -v -c 100
```