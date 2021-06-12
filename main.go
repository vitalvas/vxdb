package main

import "github.com/vitalvas/vxdb/app"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app.Execute(version, commit, date)
}
