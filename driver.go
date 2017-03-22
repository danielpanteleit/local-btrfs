package main

import (
    "fmt"
    "sync"
    "os"
    "strconv"
    "io/ioutil"
    "path"
    "encoding/json"

    "github.com/docker/go-plugins-helpers/volume"
    "github.com/fatih/color"
    "os/exec"
)

var (
    yellow = color.New(color.FgYellow).SprintfFunc()
    cyan = color.New(color.FgCyan).SprintfFunc()
    blue = color.New(color.FgBlue).SprintfFunc()
    magenta = color.New(color.FgMagenta).SprintfFunc()
    white = color.New(color.FgWhite).SprintfFunc()
)

const (
    stateDir = "/var/lib/docker/plugin-data/"
    stateFile = "local-persist.json"
)

type localPersistDriver struct {
    volumes    map[string]string
    mutex      *sync.Mutex
    debug      bool
    name       string
}

type saveData struct {
    State map[string]string `json:"state"`
}

func newLocalPersistDriver() localPersistDriver {
    fmt.Printf(white("%-18s", "Starting... "))

    driver := localPersistDriver{
        volumes : map[string]string{},
		mutex   : &sync.Mutex{},
        debug   : true,
        name    : "local-btrfs",
    }

    os.Mkdir(stateDir, 0700)

    _, driver.volumes = driver.findExistingVolumesFromStateFile()
    fmt.Printf("Found %s volumes on startup\n", yellow(strconv.Itoa(len(driver.volumes))))

    return driver
}

func (driver localPersistDriver) Get(req volume.Request) volume.Response {
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

func (driver localPersistDriver) List(req volume.Request) volume.Response {
    fmt.Print(white("%-18s", "List Called... "))

    var volumes []*volume.Volume
    for name, _ := range driver.volumes {
        volumes = append(volumes, driver.volume(name))
    }

    fmt.Printf("Found %s volumes\n", yellow(strconv.Itoa(len(volumes))))

    return volume.Response{
        Volumes: volumes,
    }
}

func (driver localPersistDriver) Create(req volume.Request) volume.Response {
    fmt.Print(white("%-18s", "Create Called... "))

    mountpoint := req.Options["mountpoint"]
    if mountpoint == "" {
        fmt.Printf("No %s option provided\n", blue("mountpoint"))
        return volume.Response{ Err: fmt.Sprintf("The `mountpoint` option is required") }
    }

    driver.mutex.Lock()
    defer driver.mutex.Unlock()

    if driver.exists(req.Name) {
        return volume.Response{ Err: fmt.Sprintf("The volume %s already exists", req.Name) }
    }

    if err := os.MkdirAll(mountpoint, 0700); err != nil {
        fmt.Printf("%17s Could not create directory %s\n", " ", magenta(mountpoint))
        return volume.Response{ Err: err.Error() }
    }
    fmt.Printf("Ensuring directory %s exists on host...\n", magenta(mountpoint))

    snapdir := mountpoint + "/snaps"
    if err := os.MkdirAll(snapdir, 0700); err != nil {
        fmt.Printf("%17s Could not create directory %s\n", " ", magenta(snapdir))
        return volume.Response{ Err: err.Error() }
    }

    filename := mountpoint + "/current"
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        cmd := exec.Command("btrfs", "subvolume", "create", filename)
        if output, err := cmd.CombinedOutput(); err != nil {
            return volume.Response{ Err: fmt.Sprintf("Could not create subvolume at %s: %s\n%s", filename, err.Error(), string(output) ) }
        }
    }

    driver.volumes[req.Name] = mountpoint
    if err := driver.saveState(driver.volumes); err != nil {
        fmt.Println(err.Error())
    }

    fmt.Printf("%17s Created volume %s with mountpoint %s\n", " ", cyan(req.Name), magenta(mountpoint))

    return volume.Response{}
}

func (driver localPersistDriver) Remove(req volume.Request) volume.Response {
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

func (driver localPersistDriver) Mount(req volume.MountRequest) volume.Response {
    fmt.Print(white("%-18s", "Mount Called... "))

    fmt.Printf("Mounted %s\n", cyan(req.Name))

    return driver.Path(volume.Request{Name: req.Name})
}

func (driver localPersistDriver) Path(req volume.Request) volume.Response {
    fmt.Print(white("%-18s", "Path Called... "))

    mpoint := driver.volumes[req.Name] + "/current"
    fmt.Printf("Returned path %s\n", magenta(mpoint))

    return volume.Response{ Mountpoint: mpoint }
}

func (driver localPersistDriver) Unmount(req volume.UnmountRequest) volume.Response {
    fmt.Print(white("%-18s", "Unmount Called... "))

    fmt.Printf("Unmounted %s\n", cyan(req.Name))

    return driver.Path(volume.Request{Name: req.Name})
}

func (driver localPersistDriver) Capabilities(req volume.Request) volume.Response {
    fmt.Print(white("%-18s", "Capabilities Called... "))

    return volume.Response{
        Capabilities: volume.Capability{ Scope: "local" },
    }
}


func (driver localPersistDriver) exists(name string) bool {
    return driver.volumes[name] != ""
}

func (driver localPersistDriver) volume(name string) *volume.Volume {
    return &volume.Volume{
        Name: name,
        Mountpoint: driver.volumes[name] + "/current",
    }
}

func (driver localPersistDriver) findExistingVolumesFromStateFile() (error, map[string]string) {
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

func (driver localPersistDriver) saveState(volumes map[string]string) error {
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
