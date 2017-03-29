package daemon

import (
    "fmt"
    "strings"
    "net/rpc"
)

type RpcApi struct {
    Driver LocalBtrfsDriver
}

func (RpcApi) Method(args *interface{}, result *interface{}) error {
    fmt.Printf("received Method call")
    return nil
}

func (api RpcApi) MakeSnap(args []string, result *string) error {
    err := api.Driver.makeSnap(args[0], args[1])
    if err != nil {
        return err
    }
    return nil
}

func (api RpcApi) ListSnaps(args []string, result *string) error {
    /*
    snaps, err := api.Driver.listSnaps(args[0])
    if err != nil {
        return err
    }
    */
    snaps := []string{"vol1", "vol2", "vol3"}


    *result = strings.Join(snaps, "\n") + "\n"

    return nil
}


type RpcApiRequest struct {
    Method string
    Args   []string
}


func MakeSnapRequest(volume string, snapname string) RpcApiRequest {
    return RpcApiRequest{
        Method: "RpcApi.MakeSnap",
        Args:   []string {volume, snapname},
    }
}
