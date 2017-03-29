package daemon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"sync"

	"errors"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/fatih/color"
	"os/exec"
	"strings"
)

var (
	yellow  = color.New(color.FgYellow).SprintfFunc()
	cyan    = color.New(color.FgCyan).SprintfFunc()
	blue    = color.New(color.FgBlue).SprintfFunc()
	magenta = color.New(color.FgMagenta).SprintfFunc()
	white   = color.New(color.FgWhite).SprintfFunc()
)

const (
	stateDir  = "/var/lib/docker/plugin-data/"
	stateFile = "local-btrfs.json"
)

type LocalBtrfsDriver struct {
	volumes map[string]string
	mutex   *sync.Mutex
	debug   bool
	Name    string
}

type saveData struct {
	State map[string]string `json:"state"`
}

func NewLocalBtrfsDriver() LocalBtrfsDriver {
	fmt.Printf(white("%-18s", "Starting... "))

	driver := LocalBtrfsDriver{
		volumes: map[string]string{},
		mutex:   &sync.Mutex{},
		debug:   true,
		Name:    "local-btrfs",
	}

	os.Mkdir(stateDir, 0700)

	_, driver.volumes = driver.findExistingVolumesFromStateFile()
	fmt.Printf("Found %s volumes on startup\n", yellow(strconv.Itoa(len(driver.volumes))))

	return driver
}

func (driver LocalBtrfsDriver) Get(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "Get Called... "))

	if driver.exists(req.Name) {
		fmt.Printf("Found %s\n", cyan(req.Name))
		return volume.Response{
			Volume: driver.volume(req.Name),
		}
	}

	fmt.Printf("Couldn't find %s\n", cyan(req.Name))
	return volume.Response{
		Err: fmt.Sprintf("No volume found with the name %s", cyan(req.Name)),
	}
}

func (driver LocalBtrfsDriver) List(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "List Called... "))

	var volumes []*volume.Volume
	for name := range driver.volumes {
		volumes = append(volumes, driver.volume(name))
	}

	fmt.Printf("Found %s volumes\n", yellow(strconv.Itoa(len(volumes))))

	return volume.Response{
		Volumes: volumes,
	}
}

func (driver LocalBtrfsDriver) Create(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "Create Called... "))

	mountpoint := req.Options["mountpoint"]
	if mountpoint == "" {
		fmt.Printf("No %s option provided\n", blue("mountpoint"))
		return volume.Response{Err: "The `mountpoint` option is required"}
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	if driver.exists(req.Name) {
		return volume.Response{Err: fmt.Sprintf("The volume %s already exists", req.Name)}
	}

	if err := os.MkdirAll(mountpoint, 0700); err != nil {
		fmt.Printf("%17s Could not create directory %s\n", " ", magenta(mountpoint))
		return volume.Response{Err: err.Error()}
	}
	fmt.Printf("Ensuring directory %s exists on host...\n", magenta(mountpoint))

	snapdir := mountpoint + "/snaps"
	if err := os.MkdirAll(snapdir, 0700); err != nil {
		fmt.Printf("%17s Could not create directory %s\n", " ", magenta(snapdir))
		return volume.Response{Err: err.Error()}
	}

	filename := mountpoint + "/current"
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := callBtrfs("subvolume", "create", filename); err != nil {
			return volume.Response{Err: err.Error()}
		}
	}

	driver.volumes[req.Name] = mountpoint
	if err := driver.saveState(driver.volumes); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("%17s Created volume %s with mountpoint %s\n", " ", cyan(req.Name), magenta(mountpoint))

	return volume.Response{}
}

func (driver LocalBtrfsDriver) Remove(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "Remove Called... "))
	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	delete(driver.volumes, req.Name)

	if err := driver.saveState(driver.volumes); err != nil {
		fmt.Println(err.Error())
	}

	fmt.Printf("Removed %s\n", cyan(req.Name))

	return volume.Response{}
}

func (driver LocalBtrfsDriver) Mount(req volume.MountRequest) volume.Response {
	fmt.Print(white("%-18s", "Mount Called... "))

	fmt.Printf("Mounted %s\n", cyan(req.Name))

	return driver.Path(volume.Request{Name: req.Name})
}

func (driver LocalBtrfsDriver) Path(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "Path Called... "))

	mpoint := driver.volumes[req.Name] + "/current"
	fmt.Printf("Returned path %s\n", magenta(mpoint))

	return volume.Response{Mountpoint: mpoint}
}

func (driver LocalBtrfsDriver) Unmount(req volume.UnmountRequest) volume.Response {
	fmt.Print(white("%-18s", "Unmount Called... "))

	fmt.Printf("Unmounted %s\n", cyan(req.Name))

	return driver.Path(volume.Request{Name: req.Name})
}

func (driver LocalBtrfsDriver) Capabilities(req volume.Request) volume.Response {
	fmt.Print(white("%-18s", "Capabilities Called... "))

	return volume.Response{
		Capabilities: volume.Capability{Scope: "local"},
	}
}

func (driver LocalBtrfsDriver) exists(name string) bool {
	return driver.volumes[name] != ""
}

func (driver LocalBtrfsDriver) volume(name string) *volume.Volume {
	return &volume.Volume{
		Name:       name,
		Mountpoint: driver.volumes[name] + "/current",
	}
}

func (driver LocalBtrfsDriver) findExistingVolumesFromStateFile() (error, map[string]string) {
	p := path.Join(stateDir, stateFile)
	fileData, err := ioutil.ReadFile(p)
	if err != nil {
		return err, map[string]string{}
	}

	var data saveData
	e := json.Unmarshal(fileData, &data)
	if e != nil {
		return e, map[string]string{}
	}

	return nil, data.State
}

func (driver LocalBtrfsDriver) saveState(volumes map[string]string) error {
	data := saveData{
		State: volumes,
	}

	fileData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	p := path.Join(stateDir, stateFile)
	return ioutil.WriteFile(p, fileData, 0600)
}

func (driver LocalBtrfsDriver) makeSnap(volumeName string, snapshotName string) error {
	// TODO: check if mounted and return warn/error

	volumePath, exists := driver.volumes[volumeName]
	if !exists {
		return errors.New("volume " + volumeName + " does not exist")
	}

	snapPath := driver.getSnapshotPath(volumePath, snapshotName)
	if _, err := os.Stat(snapPath); !os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("snapshot %q already exists for volume %q (%v)", snapshotName, volumeName, snapPath))
	}

	srcPath := volumePath + "/current"
	fmt.Printf("creating snapshot of volume %v as %v: %v -> %v\n", volumeName, snapshotName, srcPath, snapPath)
	if err := callBtrfs("subvolume", "snapshot", "-r", srcPath, snapPath); err != nil {
		return err
	}

	return nil
}

func (driver LocalBtrfsDriver) removeSnap(volumeName string, snapshotName string) error {
	// TODO: check if mounted and return warn/error

	volumePath, exists := driver.volumes[volumeName]
	if !exists {
		return errors.New("volume " + volumeName + " does not exist")
	}

	snapPath := driver.getSnapshotPath(volumePath, snapshotName)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("snapshot %q does not exist for volume %q (%v)", snapshotName, volumeName, snapPath))
	}

	fmt.Printf("removing snapshot %v of volume %v in %v\n", snapshotName, volumeName, snapPath)
	if err := callBtrfs("subvolume", "delete", snapPath); err != nil {
		return err
	}

	return nil
}

func (driver LocalBtrfsDriver) restoreSnap(volumeName string, snapshotName string) error {
	// TODO: check if mounted and return warn/error

	fmt.Printf("Restoring snapshot %v in volume %v", snapshotName, volumeName)

	volumePath, exists := driver.volumes[volumeName]
	if !exists {
		return errors.New("volume " + volumeName + " does not exist")
	}

	snapPath := driver.getSnapshotPath(volumePath, snapshotName)
	if _, err := os.Stat(snapPath); os.IsNotExist(err) {
		return errors.New(fmt.Sprintf("snapshot %q does not exist for volume %q (%v)", snapshotName, volumeName, snapPath))
	}

	currentPath := volumePath + "/current"

	fmt.Printf("removing default subvolume %v\n", currentPath)
	if err := callBtrfs("subvolume", "delete", currentPath); err != nil {
		return err
	}

	fmt.Printf("creating read-write snapshot %v -> %v\n", snapPath, currentPath)
	if err := callBtrfs("subvolume", "snapshot", snapPath, currentPath); err != nil {
		return err
	}

	return nil
}

func callBtrfs(args ...string) error {
	cmd := exec.Command("btrfs", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		msg := fmt.Sprintf("Btrfs call %v failed: %s\n%s", strings.Join(args, " "), err.Error(), string(output))
		fmt.Print(msg)
		return errors.New(msg)
	}
	return nil
}

func (driver LocalBtrfsDriver) getSnapshotPath(volumePath string, snapshotName string) string {
	return volumePath + "/snaps/" + snapshotName
}
