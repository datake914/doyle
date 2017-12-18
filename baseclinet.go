package main

import "fmt"

type baseClient struct {
}

func (c *baseClient) decorateCmd(cmd string, sudo bool) string {
	if sudo {
		cmd = fmt.Sprintf("sudo -S sh -c \"%s\"", cmd)
	}
	return cmd
}
