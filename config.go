package main

import kingpin "gopkg.in/alecthomas/kingpin.v2"

type config struct {
	targets      *[]string
	excludes     *[]string
	includes     *[]string
	tmpDir       string
	detail       *bool
	sourceServer *ServerConfig
	targetServer *ServerConfig
}

type ServerConfig struct {
	Host    *string
	Port    *string
	User    *string
	Pass    *string
	KeyPath *string
	KeyPass *string
}

func parse() *config {
	o := &config{
		targets:  kingpin.Flag("targets", "Target directories").Required().Strings(),
		excludes: kingpin.Flag("excludes", "Exclude patterns").Strings(),
		includes: kingpin.Flag("includes", "include patterns").Strings(),
		detail:   kingpin.Flag("detail", "Whether or not to display the details").Bool(),
		sourceServer: &ServerConfig{
			Host:    kingpin.Flag("srcHost", "Source server host").Required().String(),
			Port:    kingpin.Flag("srcPort", "Source server port").String(),
			User:    kingpin.Flag("srcUser", "Source server user").Required().String(),
			Pass:    kingpin.Flag("srcPass", "Source server password").String(),
			KeyPath: kingpin.Flag("srcKeyPath", "Source server key file path").String(),
			KeyPass: kingpin.Flag("srcKeyPass", "Source server key file password").String(),
		},
		targetServer: &ServerConfig{
			Host:    kingpin.Flag("tgtHost", "Target server host").Required().String(),
			Port:    kingpin.Flag("tgtPort", "Target server port").String(),
			User:    kingpin.Flag("tgtUser", "Target server user").Required().String(),
			Pass:    kingpin.Flag("tgtPass", "Target server password").String(),
			KeyPath: kingpin.Flag("tgtKeyPath", "Target server key file path").String(),
			KeyPass: kingpin.Flag("tgtKeyPass", "Target server key file password").String(),
		},
	}
	kingpin.Parse()
	return o
}
