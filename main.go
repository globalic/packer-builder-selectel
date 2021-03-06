package main

import (
  "github.com/mitchellh/packer/packer/plugin"
  "github.com/express42/packer-builder-selectel/builder/selectel"
)

func main() {

  server, err := plugin.Server()
  if err != nil {
    panic(err)
  }
  server.RegisterBuilder(new(selectel.Builder))
  server.Serve()
}
