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
	app = kingpin.New("chat", "A command-line chat application.")

	cmdDaemon = app.Command("daemon", "Register a new user.")

	cmdSnap       = app.Command("snap", "")
	cmdSnapVolume = cmdSnap.Arg("volume", "").Required().String()
	cmdSnapName   = cmdSnap.Arg("name", "").Required().String()

	/*
	   cmdSnaps       = app.Command("snaps", "")
	   cmdSnapsVolume = cmdSnaps.Arg("volume", "").String()
	*/

	//cmdRestore
	//cmdRemove
)

func Main() {
	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case cmdDaemon.FullCommand():
		serveDockerHandler()
	case cmdSnap.FullCommand():
		clientHandler(daemon.MakeSnapRequest(*cmdSnapVolume, *cmdSnapName))
		//case cmdSnaps.FullCommand():
		//    clientHandler("ListSnaps", []string{*cmdSnapsVolume})
	}
}

func serveDockerHandler() {
	driver := daemon.NewLocalBtrfsDriver()

	rpcApi := daemon.RpcApi{Driver: driver}
	rpc.Register(rpcApi)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

	handler := volume.NewHandler(driver)
	fmt.Println(handler.ServeUnix(driver.Name, 0))
}

func clientHandler(request daemon.RpcApiRequest) {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	result := new(string)
	if err := client.Call(request.Method, &request.Args, &result); err != nil {
		fmt.Printf("error: %v\n", err)
	}
	fmt.Print(*result)
}
