package main

type Client interface {
	exec(cmd string, sudo bool) (*Result, error)
	execWithPipe(cmd string, sudo bool, stdConf *stdConfig) (*Result, error)
}
