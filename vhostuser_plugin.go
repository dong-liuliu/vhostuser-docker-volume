package main

import (
	"fmt"
	//"io/ioutil"
	"log"
	"os"
	"path/filepath"

	//"strings"
	"sync"
	//"time"

	"github.com/docker/go-plugins-helpers/volume"
	"golang.org/x/sys/unix"
)

const (
	vhostUserBlk       string = "blk"
	vhostUserSCSI      string = "scsi"
	vhostUserBlkMajor  int    = 241
	vhostUserSCSIMajor int    = 242
)

type vhostUserVolume struct {
	volumeName string
	// Volume may be specified a device name different from volume name
	deviceName string
	// VhostUser type of this volume device, it can be blk, scsi and so on.
	volumeType string
	// Reference count
	mountRef int
	// Minor number of the device node
	nodeMinor int
	// volume may has a s vhostUser directory
	vPath string
}

// Use volume name as index to find vhostUserVolume
var volumes = make(map[string]vhostUserVolume)
var m = &sync.Mutex{}
var blkNodeMinor typeNodeMinor
var scsiNodeMinor typeNodeMinor

// Each time, docker plugin helper will pass a new created plugin object copied
// from firstly created plugin handler, so move context into global variables
type vhostUserPlugin struct {
	vPath string
}

func createVhostUserSubDir(vhostUserDir string) error {
	var err error

	err = os.MkdirAll(vhostUserDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(vhostUserDir, "block"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(vhostUserDir, "block", "sockets"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Join(vhostUserDir, "block", "devices"), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func IsExistedDir(path string) bool {
	f, err := os.Stat(path)
	if err != nil || !f.IsDir() {
		return false
	}

	return true
}

func checkVhostUserDir(vhostUserDir string) error {
	if !IsExistedDir(vhostUserDir) ||
		!IsExistedDir(filepath.Join(vhostUserDir, "block")) ||
		!IsExistedDir(filepath.Join(vhostUserDir, "block", "sockets")) ||
		!IsExistedDir(filepath.Join(vhostUserDir, "block", "devices")) {
		return fmt.Errorf(vhostUserDir + " is not a valid vhost-user directory")
	}

	return nil
}

func newVhostUserPlugin(vhostUserPath string) (*vhostUserPlugin, error) {
	log.Printf("Start to create vhost-user plugin")

	// mkdir for vhost-user device if required dir doesn't exist
	err := createVhostUserSubDir(vhostUserPath)
	if err != nil {
		log.Printf("Failed at creating sub-directory %s: %s", vhostUserPath, err.Error())
		return nil, err
	}

	vPlugin := vhostUserPlugin{
		vPath: vhostUserPath,
	}

	return &vPlugin, nil
}

func (vPlugin vhostUserPlugin) Create(request *volume.CreateRequest) error {
	m.Lock()
	defer m.Unlock()
	log.Printf("Create volume %v", *request)

	var err error
	volumeName := request.Name

	// Check volume type
	volumeType := vhostUserBlk
	if _, ok := request.Options["type"]; ok {
		switch request.Options["type"] {
		case "blk", "BLK":
		case "scsi", "SCSI":
			volumeType = vhostUserSCSI
		default:
			return fmt.Errorf("Unknown volume type %s", request.Options["type"])
		}
	}

	// Check backend device name
	deviceName := volumeName
	if _, ok := request.Options["device"]; ok {
		deviceName = request.Options["device"]
	}

	// Check whether it has a specific vhost-user dir
	vPath := vPlugin.vPath
	if _, ok := request.Options["path"]; ok {
		vPath = request.Options["path"]

		err = checkVhostUserDir(vPath)
		if err != nil {
			return err
		}
	}

	// check existence of this named volume and its type
	if _, ok := volumes[volumeName]; ok {
		return fmt.Errorf("volume is already created")
	}

	// Make sure there is no volume whose has a same device name
	socketPath := filepath.Join(vPath, "block", "sockets", deviceName)
	for _, vVolume := range volumes {
		existedSocket := filepath.Join(vVolume.vPath, "block", "sockets", vVolume.deviceName)
		if existedSocket == socketPath {
			return fmt.Errorf("device is already referenced by volume " + vVolume.volumeName)
		}
	}

	// check existence of this volume's device socket
	_, err = os.Stat(socketPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("Unix socket %s for volume is not exist", socketPath)
	}

	// mknod for this volume device
	nodePath := filepath.Join(vPath, "block", "devices", deviceName)

	var nodeMajor, nodeMinor int
	switch volumeType {
	case vhostUserBlk:
		nodeMajor = vhostUserBlkMajor
		nodeMinor, err = blkNodeMinor.getNodeMinor()
	case vhostUserSCSI:
		nodeMajor = vhostUserSCSIMajor
		nodeMinor, err = scsiNodeMinor.getNodeMinor()
	}
	if err != nil {
		return err
	}

	err = unix.Mknod(nodePath, unix.S_IFBLK, int(unix.Mkdev(uint32(nodeMajor), uint32(nodeMinor))))
	if err != nil {
		log.Printf("Failed at creating device node file %s: %s", nodePath, err.Error())
		return err
	}

	newVolume := vhostUserVolume{
		volumeName: volumeName,
		deviceName: deviceName,
		volumeType: volumeType,
		mountRef:   0,
		nodeMinor:  nodeMinor,
		vPath:      vPath,
	}

	volumes[volumeName] = newVolume

	return nil
}

func (vPlugin vhostUserPlugin) Remove(request *volume.RemoveRequest) error {
	m.Lock()
	defer m.Unlock()
	log.Printf("Remove volume %s", request.Name)

	// Check existence of this named volume
	volumeName := request.Name
	if _, ok := volumes[volumeName]; !ok {
		return fmt.Errorf("volume doesn't exist")
	}

	// check mount reference count, whether it is 0
	vVolume := volumes[volumeName]
	//assert(vVolume.mountRef >= 0)
	if vVolume.mountRef > 0 {
		return fmt.Errorf("volume is still in use")
	}

	var err error
	switch vVolume.volumeType {
	case vhostUserBlk:
		err = blkNodeMinor.putNodeMinor(vVolume.nodeMinor)
	case vhostUserSCSI:
		err = scsiNodeMinor.putNodeMinor(vVolume.nodeMinor)
	}
	if err != nil {
		return err
	}

	// delete the node made by make node
	nodePath := filepath.Join(vVolume.vPath, "block", "devices", vVolume.deviceName)
	err = os.Remove(nodePath)
	if err != nil {
		log.Printf("Failed at removing device node file %s: %s", nodePath, err.Error())
		return err
	}

	delete(volumes, volumeName)
	log.Printf("volume %s is removed", volumeName)

	return nil
}

func (vPlugin vhostUserPlugin) Get(request *volume.GetRequest) (*volume.GetResponse, error) {
	log.Printf("Get volume information %s", request.Name)

	rVolume := volume.Volume{}
	response := &volume.GetResponse{Volume: &rVolume}

	// Check existence of this named volume
	volumeName := request.Name
	if _, ok := volumes[volumeName]; !ok {
		return response, fmt.Errorf("volume doesn't exist")
	}

	vVolume := volumes[volumeName]
	nodePath := filepath.Join(vVolume.vPath, "block", "devices", vVolume.deviceName)
	response.Volume.Name = volumeName
	response.Volume.Mountpoint = nodePath
	//TODO add more info into response.Volume.Status[]

	return response, nil
}

func (vPlugin vhostUserPlugin) List() (*volume.ListResponse, error) {
	m.Lock()
	defer m.Unlock()
	log.Printf("List volumes")

	var vVolumes []*volume.Volume
	response := &volume.ListResponse{}

	for _, vVolume := range volumes {
		vVolumes = append(vVolumes, &volume.Volume{Name: vVolume.volumeName, Mountpoint: vVolume.vPath})
	}

	response.Volumes = vVolumes
	return response, nil
}

func (vPlugin vhostUserPlugin) Mount(request *volume.MountRequest) (*volume.MountResponse, error) {
	m.Lock()
	defer m.Unlock()
	log.Printf("mount volume %s to %s", request.Name, request.ID)

	response := &volume.MountResponse{Mountpoint: ""}

	// Check existence of this named volume
	volumeName := request.Name
	if _, ok := volumes[volumeName]; !ok {
		return response, fmt.Errorf("volume doesn't exist")
	}

	// increase its mount reference count
	vVolume := volumes[volumeName]
	vVolume.mountRef++
	nodePath := filepath.Join(vVolume.vPath, "block", "devices", vVolume.deviceName)
	response.Mountpoint = nodePath

	return response, nil
}

func (vPlugin vhostUserPlugin) Path(request *volume.PathRequest) (*volume.PathResponse, error) {
	log.Printf("Get volume path %s", request.Name)
	response := &volume.PathResponse{Mountpoint: ""}

	// Check existence of this named volume
	volumeName := request.Name
	if _, ok := volumes[volumeName]; !ok {
		return response, fmt.Errorf("volume doesn't exist")
	}

	vVolume := volumes[volumeName]
	nodePath := filepath.Join(vVolume.vPath, "block", "devices", vVolume.deviceName)
	response.Mountpoint = nodePath

	return response, nil
}

func (vPlugin vhostUserPlugin) Unmount(request *volume.UnmountRequest) error {
	m.Lock()
	defer m.Unlock()
	log.Printf("Unmount volume %s from %s", request.Name, request.ID)

	// Check existence of this named volume
	volumeName := request.Name
	if _, ok := volumes[volumeName]; !ok {
		return fmt.Errorf("volume doesn't exist")
	}

	// decrease its mount reference count
	vVolume := volumes[volumeName]
	if vVolume.mountRef < 1 {
		return fmt.Errorf("volume is not mounted")
	}
	vVolume.mountRef--

	return nil
}

func (vPlugin vhostUserPlugin) Capabilities() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{Capabilities: volume.Capability{Scope: "local"}}
}
