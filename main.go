package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/docker/go-plugins-helpers/volume"
)

const (
	dockerPluginDir     string = "/run/docker/plugins"
	defaultVhostUserDir string = "/var/run/kata-containers/vhost-user"
	pluginName          string = "vhostuser-docker-volume"
)

var (
	Version   string
	BuildTime string
)

func main() {

	vhostUserPath := flag.String("path", defaultVhostUserDir, "Directory of vhost-user device")
	showVersion := flag.Bool("version", false, "Show version and build time")
	flag.Parse()

	if *showVersion {
		fmt.Printf("\nVersion: %s --- BuildTime: %s\n", Version, BuildTime)
		return
	}

	// Remove unix-socket if plugin is killed
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		s := <-c
		syscall.Unlink(filepath.Join(dockerPluginDir, pluginName+".sock"))
		fmt.Println("Exit on signal ", s)

		os.Exit(0)
	}()

	vPlugin, err := newVhostUserPlugin(*vhostUserPath)
	if err != nil {
		log.Print("Failed to create vhost-user plugin")
		return
	}
	handler := volume.NewHandler(vPlugin)

	u, _ := user.Lookup("root")
	gid, _ := strconv.Atoi(u.Gid)
	fmt.Println(handler.ServeUnix(pluginName, gid))
	if err != nil {
		log.Print("Failed to listen on socket")
		return
	}
}
