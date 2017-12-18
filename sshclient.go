package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/ssh"
)

type sshClient struct {
	baseClient
	client *ssh.Client
	conf   *ServerConfig
}

type stdConfig struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func NewSshClient(conf *ServerConfig) (*sshClient, error) {
	c := new(sshClient)
	c.conf = conf
	// Create Authentication methods.
	auths := make([]ssh.AuthMethod, 0, 2)
	auths = c.addKeyAuth(auths, *conf.KeyPath)
	auths = c.addPasswordAuth(auths, *conf.Pass)
	// Create ssh client config.
	sshConf := &ssh.ClientConfig{
		User:            *conf.User,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	// Create client.
	client, err := ssh.Dial("tcp", *conf.Host+":"+*conf.Port, sshConf)
	c.client = client
	return c, err
}

func (c *sshClient) exec(cmd string, sudo bool) (*Result, error) {
	return c.execWithPipe(cmd, sudo, &stdConfig{})
}

func (c *sshClient) execWithPipe(cmd string, sudo bool, stdConf *stdConfig) (*Result, error) {
	// Create result.
	result := &Result{
		Host: *c.conf.Host,
		Port: *c.conf.Port,
	}

	// Open a new Session.
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create a new session: %v\n", err)
	}
	defer session.Close()

	// Create pty.
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, fmt.Errorf("request for pseudo terminal failed: %s", err)
	}

	// Set custom standart in/out.
	var stdout, stderr io.Writer
	if stdConf.Stdout != nil {
		stdout = stdConf.Stdout
	} else {
		stdout = new(bytes.Buffer)
	}
	if stdConf.Stderr != nil {
		stderr = stdConf.Stderr
	} else {
		stderr = new(bytes.Buffer)
	}
	session.Stdout = stdout
	session.Stderr = stderr

	// Run command.
	cmd = c.decorateCmd(cmd, sudo)
	result.Cmd = cmd
	if err = session.Run(cmd); err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			result.ExitStatus = exitErr.ExitStatus()
		} else {
			return result, fmt.Errorf("command[%s] execution failed: %s", cmd, err)
		}
	} else {
		result.ExitStatus = 0
	}

	// Set exec result.
	if buf, ok := stdout.(*bytes.Buffer); ok {
		result.Stdout = buf.String()
	}
	if buf, ok := stderr.(*bytes.Buffer); ok {
		result.Stderr = buf.String()
	}
	return result, nil
}

func (c *sshClient) addKeyAuth(auths []ssh.AuthMethod, path string) []ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(path)
	if err != nil {
		return nil
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return append(auths, ssh.PublicKeys(key))
}

func (c *sshClient) addPasswordAuth(auths []ssh.AuthMethod, pass string) []ssh.AuthMethod {
	return append(auths, ssh.Password(pass))
}
