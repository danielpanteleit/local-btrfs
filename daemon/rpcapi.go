package daemon

import (
	"strings"
    "strconv"
    "fmt"
)

type RpcApi struct {
	Driver LocalBtrfsDriver
}

func (api RpcApi) CreateVolume(args []string, result *string) error {
    return api.Driver.createVolume(args[0], args[1])
}

func (api RpcApi) RemoveVolume(args []string, result *string) error {
    purge, err := strconv.ParseBool(args[1])
    if err != nil {
        return err
    }
    return api.Driver.removeVolume(args[0], purge)
}

func (api RpcApi) CreateSnap(args []string, result *string) error {
	return api.Driver.createSnap(args[0], args[1])
}

func (api RpcApi) ListSnapshots(args []string, result *string) error {
    snaps, err := api.Driver.listSnapshots(args[0])
    if err != nil {
        return err
    }

    if len(snaps) > 0 {
        *result = strings.Join(snaps, "\n") + "\n"
    }

    return nil
}

func (api RpcApi) RemoveSnap(args []string, result *string) error {
    return api.Driver.removeSnap(args[0], args[1])
}

func (api RpcApi) RestoreSnap(args []string, result *string) error {
    return api.Driver.restoreSnap(args[0], args[1])
}

type RpcApiRequest struct {
	Method string
	Args   []string
}

func CreateVolumeRequest(volume string, path string) RpcApiRequest {
    return RpcApiRequest{"RpcApi.CreateVolume",[]string{volume, path}}
}

func RemoveVolumeRequest(volume string, purge bool) RpcApiRequest {
    return RpcApiRequest{"RpcApi.RemoveVolume",[]string{volume, fmt.Sprintf("%v", purge)}}
}

func CreateSnapRequest(volume string, snapname string) RpcApiRequest {
	return RpcApiRequest{"RpcApi.CreateSnap",[]string{volume, snapname}}
}

func ListSnapshotsRequest(volume string) RpcApiRequest {
    return RpcApiRequest{"RpcApi.ListSnapshots",[]string{volume}}
}

func RemoveSnapRequest(volume string, snapshot string) RpcApiRequest {
    return RpcApiRequest{"RpcApi.RemoveSnap",[]string{volume, snapshot}}
}

func RestoreSnapRequest(volume string, snapshot string) RpcApiRequest {
    return RpcApiRequest{"RpcApi.RestoreSnap",[]string{volume, snapshot}}
}
