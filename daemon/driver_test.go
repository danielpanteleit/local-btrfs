package daemon

import (
	"os"
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
	"os/exec"
)

var (
	defaultTestName       = "test-volume"
	defaultTestMountpoint = "/btrfs/local-btrfs-test"
)

func TestCreate(t *testing.T) {
	driver := NewLocalBtrfsDriver()

	defaultCreateHelper(driver, t)

	// test that a directory is created
	_, err := os.Stat(defaultTestMountpoint)
	if os.IsNotExist(err) {
		t.Error("Mountpoint directory was not created:", err.Error())
	}

	// test that volumes has one
	if len(driver.volumes) != 1 {
		t.Error("Driver should have exactly 1 volume")
	}

	defaultCleanupHelper(driver, t)

	// test that options are required
	res := driver.Create(volume.Request{
		Name: defaultTestName,
	})

	if res.Err != "The `mountpoint` option is required" {
		t.Error("Should error out without mountpoint option")
	}
}

func TestGet(t *testing.T) {
	driver := NewLocalBtrfsDriver()

	defaultCreateHelper(driver, t)

	res := driver.Get(volume.Request{Name: defaultTestName})
	if res.Err != "" {
		t.Error("Should have found a volume!")
	}

	defaultCleanupHelper(driver, t)
}

func TestList(t *testing.T) {
	driver := NewLocalBtrfsDriver()

	name := defaultTestName + "2"
	mountpoint := defaultTestMountpoint + "2"

	defaultCreateHelper(driver, t)
	res := driver.List(volume.Request{})
	if len(res.Volumes) != 1 {
		t.Error("Should have found 1 volume!")
	}

	createHelper(driver, t, name, mountpoint)
	res2 := driver.List(volume.Request{})
	if len(res2.Volumes) != 2 {
		t.Error("Should have found 1 volume!")
	}

	defaultCleanupHelper(driver, t)
	cleanupHelper(driver, t, name, mountpoint)
}

func TestMountUnmountPath(t *testing.T) {
	driver := NewLocalBtrfsDriver()

	defaultCreateHelper(driver, t)

	// mount, mount and path should have same output (they all use Path under the hood)
	pathRes := driver.Path(volume.Request{Name: defaultTestName})
	mountRes := driver.Mount(volume.MountRequest{Name: defaultTestName})
	unmountRes := driver.Unmount(volume.UnmountRequest{Name: defaultTestName})

	if !(pathRes.Mountpoint == mountRes.Mountpoint &&
		mountRes.Mountpoint == unmountRes.Mountpoint &&
		unmountRes.Mountpoint == defaultTestMountpoint+"/current") {
		t.Error("Mount, Unmount and Path should all return the same Mountpoint")
	}
}

func createHelper(driver LocalBtrfsDriver, t *testing.T, name string, mountpoint string) {
	res := driver.Create(volume.Request{
		Name: name,
		Options: map[string]string{
			"mountpoint": mountpoint,
		},
	})

	if res.Err != "" {
		t.Error(res.Err)
	}
}

func defaultCreateHelper(driver LocalBtrfsDriver, t *testing.T) {
	createHelper(driver, t, defaultTestName, defaultTestMountpoint)
}

func cleanupHelper(driver LocalBtrfsDriver, t *testing.T, name string, mountpoint string) {
	if _, err := os.Stat(defaultTestMountpoint); !os.IsNotExist(err) {
		cmd := exec.Command("../scripts/test-cleanup.sh", defaultTestMountpoint)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Error("[Cleanup] Error removing all subvolumes from test directory", err.Error(), "\n", string(output))
		}
	}

	os.RemoveAll(mountpoint)

	if _, err := os.Stat(mountpoint); !os.IsNotExist(err) {
		if err != nil {
			t.Error("[Cleanup] Error checking if mountpoint still exists", err.Error())
		} else {
			t.Error("[Cleanup] Mountpoint still exists")
		}
	}

	driver.Remove(volume.Request{Name: name})

	res := driver.Get(volume.Request{Name: name})
	if res.Err == "" {
		t.Error("[Cleanup] Volume still exists:", res.Err)
	}
}

func defaultCleanupHelper(driver LocalBtrfsDriver, t *testing.T) {
	cleanupHelper(driver, t, defaultTestName, defaultTestMountpoint)
}
