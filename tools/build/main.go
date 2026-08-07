package main

import (
	"github.com/RayleaBot/RayleaBot/sdk/go/pluginbuild"
	"github.com/RayleaBot/RayleaBot/sdk/go/pluginbuild/buildcmd"
)

func main() {
	buildcmd.Main(buildcmd.Config{
		BackendPackage: "./cmd/fortune",
		MappedAssets: []pluginbuild.AssetMapping{{
			Source: "internal/assets/fortunes.json", Destination: "fortunes.json",
		}},
	})
}
