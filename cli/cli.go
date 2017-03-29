package cli

import (
	"fmt"
	"github.com/danielpanteleit/local-btrfs/daemon"
	"github.com/docker/go-plugins-helpers/volume"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

var (
	app = kingpin.New("local-btrfs", "")

	daemonCmd = app.Command("daemon", "Starts the daemon.")

    addCmd = app.Command("add", "Adds a volume")
    addArgVolume = addCmd.Arg("volume", "").Required().String()
    addArgPath = addCmd.Arg("path", "").Required().String()

    rmCmd = app.Command("rm", "Removes volume")
    rmForceFlag = rmCmd.Flag("purge", "Removes the volume on disk").Short('p').Bool()
    rmArgVolume = rmCmd.Arg("volume", "").Required().String()

    pathCmd = app.Command("path", "Shows path to the volume")

	snapCmd = app.Command("snap", "Manages snapshots")

    snapAddCmd       = snapCmd.Command("add", "")
	snapAddArgVolume = snapAddCmd.Arg("volume", "").Required().String()
	snapAddArgName   = snapAddCmd.Arg("name", "").Required().String()

    snapLsCmd       = snapCmd.Command("ls", "")
    snapLsArgVolume = snapLsCmd.Arg("volume", "").Required().String()

    snapRmCmd       = snapCmd.Command("rm", "")
    snapRmArgVolume = snapRmCmd.Arg("volume", "").Required().String()
    snapRmArgName   = snapRmCmd.Arg("name", "").Required().String()

    snapRestoreCmd       = snapCmd.Command("restore", "")
    snapRestoreArgVolume = snapRestoreCmd.Arg("volume", "").Required().String()
    snapRestoreArgName   = snapRestoreCmd.Arg("name", "").Required().String()
)

func Main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case daemonCmd.FullCommand():
		runDaemon()
    case addCmd.FullCommand():
        clientHandler(daemon.CreateVolumeRequest(*addArgVolume, *addArgPath))
    case rmCmd.FullCommand():
        clientHandler(daemon.RemoveVolumeRequest(*rmArgVolume, *rmForceFlag))
    case pathCmd.FullCommand():
        fmt.Printf("not implemented yet!\n")
	case snapAddCmd.FullCommand():
		clientHandler(daemon.CreateSnapRequest(*snapAddArgVolume, *snapAddArgName))
    case snapLsCmd.FullCommand():
        clientHandler(daemon.ListSnapshotsRequest(*snapLsArgVolume))
    case snapRmCmd.FullCommand():
        clientHandler(daemon.RemoveSnapRequest(*snapRmArgVolume, *snapRmArgName))
    case snapRestoreCmd.FullCommand():
        clientHandler(daemon.RestoreSnapRequest(*snapRestoreArgVolume, *snapRestoreArgName))
	}
}

func runDaemon() {
	driver := daemon.NewLocalBtrfsDriver()

    setupRpcHandler(driver)

    handler := volume.NewHandler(driver)
    fmt.Println(handler.ServeUnix(driver.Name, 0))
}

func setupRpcHandler(driver daemon.LocalBtrfsDriver) {
    rpcApi := daemon.RpcApi{Driver: driver}
    rpc.Register(rpcApi)
    rpc.HandleHTTP()

    sockFile := "/var/run/local-btrfs.sock"
    l, e := net.Listen("unix", sockFile)
    if e != nil {
        log.Fatal("listen error:", e)
    }

    if err := os.Chmod(sockFile, 0700); err != nil {
        log.Fatal(err)
    }
    
    go http.Serve(l, nil)
}

func clientHandler(request daemon.RpcApiRequest) {
	client, err := rpc.DialHTTP("unix", "/var/run/local-btrfs.sock")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	result := new(string)
	if err := client.Call(request.Method, &request.Args, &result); err != nil {
		fmt.Printf("error: %v\n", err)
	}
	fmt.Print(*result)
}
