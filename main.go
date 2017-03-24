package main

import (
    "fmt"

    "github.com/docker/go-plugins-helpers/volume"
)

func main() {
    driver := newLocalBtrfsDriver()

    handler := volume.NewHandler(driver)
    fmt.Println(handler.ServeUnix(driver.name, 0))
}
