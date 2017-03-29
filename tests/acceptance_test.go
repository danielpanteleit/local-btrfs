package tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	cli     = "../bin/linux/amd64/local-btrfs"
	datadir = "/btrfs/acceptance"
)

var (
	volnum = 1
)

func newVolName() string {
	volname := fmt.Sprintf("myvol%d", volnum)
	volnum += 1
	return volname
}

func startDaemon() *exec.Cmd {
	cmd := exec.Command(cli, "daemon")
	if err := cmd.Start(); err != nil {
		log.Fatal("failed to start daemon", err)
	}

	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if fileExists("/var/run/local-btrfs.sock") {
			break
		}
	}

	return cmd
}

func stopDaemon(cmd *exec.Cmd) {
	if err := cmd.Process.Kill(); err != nil {
		log.Fatal("failed to kill: ", err)
	}
	if err := os.Remove("/var/run/local-btrfs.sock"); err != nil {
		log.Fatal("failed to remove socket: ", err)
	}
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func run(args ...string) string {
	cmd := exec.Command(cli, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("CLI call %v failed: %s\n%s", strings.Join(args, " "), err.Error(), string(output))
		panic("failed")
	}
	return string(output)
}

func volumePath(volume string) string {
	return datadir + "/" + volume
}

func currentPath(volume string) string {
	return volumePath(volume) + "/current"
}

func Test_add_createsVolume(t *testing.T) {
	defer stopDaemon(startDaemon())

	volume := "myvol1"
	voldir := volumePath(volume)
	run("add", volume, voldir)

	if _, err := os.Stat(currentPath(volume)); os.IsNotExist(err) {
		t.Fatal("did not create volume directory")
	}
}

func Test_rm_doesNotRemoveVolume(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()

	run("rm", volume)

	if _, err := os.Stat(currentPath(volume)); os.IsNotExist(err) {
		t.Fatal("volume directory does not exist")
	}
}

func Test_rm_removesVolume_withPurge(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()

	run("rm", "--purge", volume)

	if _, err := os.Stat(volumePath(volume)); !os.IsNotExist(err) {
		t.Fatal("volume directory does exist")
	}
}

func Test_rm_removesVolume_withPurge_includingSnapshots(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()
	ioutil.WriteFile(currentPath(volume)+"/someFile", []byte("Some content"), 0644)
	run("snap", "add", volume, "snap1")

	run("rm", "--purge", volume)

	if _, err := os.Stat(volumePath(volume)); !os.IsNotExist(err) {
		t.Fatal("volume directory does exist")
	}
}

func Test_snapLs_listsNothing_withNoSnapshots(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()
	defer removeVolume(volume)

	result := run("snap", "ls", volume)

	assert.Equal(t, "", result)
}

func Test_snapLs_listsSnapshots(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()
	defer removeVolume(volume)

	run("snap", "add", volume, "snap1")
	run("snap", "add", volume, "snap2")
	result := run("snap", "ls", volume)

	assert.Equal(t, "snap1\nsnap2\n", result)
}

func removeFile(path string) {
	if err := os.Remove(path); err != nil {
		panic(err.Error())
	}
}

func createVolume() string {
	volume := newVolName()
	voldir := volumePath(volume)
	run("add", volume, voldir)

	return volume
}

func removeVolume(volume string) {
	run("rm", "--purge", volume)
}

func Test_snapRestore_restoresSnapshots(t *testing.T) {
	defer stopDaemon(startDaemon())
	volume := createVolume()
	defer removeVolume(volume)

	path := currentPath(volume) + "/someFile"
	content := []byte("Some content")
	ioutil.WriteFile(path, content, 0644)

	run("snap", "add", volume, "snap")

	removeFile(path)

	run("snap", "restore", volume, "snap")

	actual, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("could not read file")
	}
	assert.Equal(t, content, actual)
}
