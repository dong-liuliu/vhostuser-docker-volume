# vhost-user volume plug-in for Docker

This plugin allows you to use vhost-user storage device with Docker. The plugin is intended to run docker with vhost-user device as volume without assigning a long path, also without manually creating device node.

This plugin is a sample and it doesn't contain any operations to the vhost-user target. So users should operate their vhost-user target application directly. The vhost-user target can be SPDK vhost-user target, or other targets like vdpa based smart NIC.

It can be used by docker with Kata Container Runtime

## Build

Using `./build.sh` to build this plugin can generate version info and build time into plugin.

## Setup

Note: Here, we take path `/var/run/kata-containers/vhost-user/` as the vhost-user directory for container volume usage.

### vhost-user volume plugin setup

Download vhost-user volume plugin from (http://github.com/dong-liuliu/vhostuser-docker-volume).
Run the plugin also with the vhost-user directory parameter.

```bash
./vhostuser-docker-volume -p /var/run/kata-containers/vhost-user/ &
```

Then the volume plugin will create the vhost-user directory and related sub-directories:

```
/var/run/kata-containers/vhost-user/
/var/run/kata-containers/vhost-user/block/
/var/run/kata-containers/vhost-user/block/sockets/
/var/run/kata-containers/vhost-user/block/devices/
```

Also one UNIX socket will be created under `/run/docker/plugins/` to represent itself to docker. The full pathname for the socket is:

```
/run/docker/plugins/vhostuser-docker-volume.sock
```

### Vhost-user Target Application Setup

As described previously, vhost-user target application is not controlled by this plugin. So here use SPDK vhost-user target as a reference to operate.

- Download and compile SPDK following [SPDK](https://spdk.io) and [SPDK vhost-user target](https://spdk.io/doc/vhost.html).

- setup env and start SPDK target by:
  
  ```bash
  $ sudo HUGEMEM=4096 scripts/setup.sh
  $ sudo app/spdk_tgt/spdk_tgt -S /var/run/kata-containers/vhost-user/block/sockets/ &
  ```

- Create 3 memory based vhost-user-blk devices for later exercise
  
  ```bash
  $ sudo scripts/rpc.py bdev_malloc_create 64 4096 -b Malloc0
  $ sudo scripts/rpc.py bdev_malloc_create 64 4096 -b Malloc1
  $ sudo scripts/rpc.py bdev_malloc_create 64 4096 -b Malloc2
  $ sudo scripts/rpc.py vhost_create_blk_controller vhostblk0 Malloc0
  $ sudo scripts/rpc.py vhost_create_blk_controller vhostblk1 Malloc1
  $ sudo scripts/rpc.py vhost_create_blk_controller vhostblk2 Malloc2
  ```

## Usage

### Commands explanation for vhost-user docker volume plugin

When start this plugin you can specify vhost-user path by `-path`, like

```bash
./vhostuser-docker-volume -path <vhost-user directory path>
```

When create volume in vhost-user type device, there are 3 options for user.

- `-o type=<device type>`
  Specify the vhost-user device type, it can be "blk" for vhost-user-blk device or "scsi" for "vhost-user-scsi" device. By default, it is "blk".
- `-o device=<device name>`
  Spcecify the vhost-user device name. By default, it is same with the volume name
- `-o path=<another vhost-user path>`
  Specify a different vhost-user path for the new volume with this plugin's vhost-user volume.

A typical create volume command with full expansion is:

```bash
docker volume create -d vhostuser-docker-volume vhostblk0 -o type=blk -o device=vhostblk0 -o path=/var/run/kata-containers/vhost-user
```

For short, it is

```bash
docker volume create -d vhostuser-docker-volume vhostblk0
```

### Examples

With previous setup on vhostuser-docker-plugin and spdk vhost-user target, let's do some operations on the volumes and run docker with one of them.

- Create 3 volumes
  
  ```bash
  $ docker volume create --driver vhostuser-docker-volume --name volume0 --opt device=vhostblk0 --opt type=blk
  $ docker volume create --driver vhostuser-docker-volume --name volume1 --opt device=vhostblk1 --opt type=blk
  $ docker volume create --driver vhostuser-docker-volume --name volume2 --opt device=vhostblk2 --opt type=blk
  ```

- Delete a volume

```bash
$ docker volume rm volume1
```

- List all volumes

```bash
$ docker volume ls
```

- Inspect a volume

```bash
$ docker volume inspect volume0
```

- Run docker in Kata runtime with one vhost-user volume

```bash
$ docker run --runtime kata-runtime --volume-driver=vhostuser-docker-volume -v volume0:/data -ti busybox sh
```
