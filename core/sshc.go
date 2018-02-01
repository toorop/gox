package core

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient is a SSH client Waooo
type SSHClient struct {
	conn  *ssh.Client
	pipes []string
}

// NewSSHClient return a SSH client
func NewSSHClient(host, user, keyFile string, pipes []string) (client *SSHClient, err error) {
	// Auth method
	buffer, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return client, err
	}
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return client, err
	}

	// config
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(key)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		return nil, err
	}
	return &SSHClient{
		conn:  conn,
		pipes: pipes,
	}, nil
}

// GetSession return a new session
func (c *SSHClient) GetSession() (session *ssh.Session, err error) {
	session, err = c.conn.NewSession()
	//return
	if err != nil {
		return
	}
	for _, pipe := range c.pipes {
		switch pipe {
		case "stdin":
			session.Stdin = os.Stdin
		case "stdout":
			session.Stdout = os.Stdout
		case "stderr":
			session.Stderr = os.Stderr
		}
	}
	return session, nil
}

// CopyFile copy file
func (c *SSHClient) CopyFile(localPath, remotePath, perm string) error {
	// get session
	session, err := c.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Local file
	fd, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer fd.Close()
	info, err := fd.Stat()
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	go func() {
		var writter io.WriteCloser
		writter, err = session.StdinPipe()
		if err != nil {
			errChan <- err
			return
		}
		defer writter.Close()
		fmt.Fprintln(writter, "C"+perm, info.Size(), path.Base(remotePath))
		_, err = io.Copy(writter, fd)
		if err != nil {
			errChan <- err
			return
		}
		fmt.Fprint(writter, "\x00")
		errChan <- nil
	}()

	if err = session.Run("/usr/bin/scp -t " + remotePath); err != nil {
		return err
	}

	// handle err from go routine
	select {
	case err = <-errChan:
		return err
	}
}

// Run run cmd
func (c *SSHClient) Run(cmd string) error {
	session, err := c.GetSession()
	if err != nil {
		return err
	}
	defer session.Close()
	return session.Run(cmd)
}

// RunOrDie runc cmd, exit on failure
func (c *SSHClient) RunOrDie(cmd string) {
	if err := c.Run(cmd); err != nil {
		log.Fatalln("ERR - unable to run ", cmd, ". ", err)
	}
}

// GetOutput rerurn command output
func (c *SSHClient) GetOutput(cmd string) (output []byte, err error) {
	s, err := c.GetSession()
	if err != nil {
		return output, err
	}
	defer s.Close()
	out, err := s.StdoutPipe()
	if err != nil {
		return output, err
	}
	if err = s.Run(cmd); err != nil {
		return
	}
	return ioutil.ReadAll(out)
}
