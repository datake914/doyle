package main

type Result struct {
	Host       string
	Port       string
	Cmd        string
	Stdout     string
	StdoutPath string
	Stderr     string
	StderrPath string
	ExitStatus int
	err        error
}
