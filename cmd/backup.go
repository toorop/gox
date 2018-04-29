// Copyright © 2018 Stéphane Depierrepont
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/toorop/gox/core"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup a remote MySQL server",
	Long: `Backup a remote MySQL server

	gox --backup --config path/to/config.yaml

Example:
	gox --backup --config server1.yaml
	`,
	Run: backup,
}

func init() {
	rootCmd.AddCommand(backupCmd)
}

// run backup
//ssh  xtrabackup --host --user -password --compress --compress-threads --galera-info --stream=xbstream target-dir=/tmp
func backup(cmd *cobra.Command, args []string) {

	// SSH
	// TODO check required
	sshUser := viper.GetString("ssh.user")
	sshKey := viper.GetString("ssh.key")

	// command
	command := viper.GetString("xtrabackup")
	if command == "" {
		command = "xtrabackup --"
	}

	command += " --backup --stream=xbstream"

	// host
	host := viper.GetString("host")
	if host == "" {
		log.Fatalln("ERR - 'host' not found in config")
	}

	// user
	user := viper.GetString("dbuser")
	if user == "" {
		log.Fatalln("ERR - 'user' not found in config")
	}

	// target
	backupDir := viper.GetString("backup-dir")
	if backupDir == "" {
		log.Fatalln("ERR - 'backup-dir' not found in config")
	}

	command = fmt.Sprintf("%s --user=%s", command, user)

	if viper.GetString("dbpassword") != "" {
		command += " --password=\"" + viper.GetString("dbpassword") + "\""
	}

	// Compress ?
	if viper.GetBool("compress.active") {
		command += " --compress"
		if viper.GetInt("compress.threads") != 0 {
			command = fmt.Sprintf("%s --compress-threads %d", command, viper.GetInt("compress.threads"))
		}
	}

	// Galera ?
	if viper.GetBool("galera") {
		command += " --galera-info"
	}

	// keep
	keep := viper.GetString("keep")

	// Open target file
	targetPath := path.Join(backupDir, time.Now().Format("2006-01-02--15-04-05"))
	if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
		log.Fatalln("ERR - unable to create folder "+targetPath+" : ", err)
	}
	fileWritter, err := os.Create(path.Join(targetPath, "backup.xbstream"))
	if err != nil {
		log.Fatalln("ERR - unable to create "+path.Join(targetPath, "backup.xbstream")+" -", err)
	}
	defer fileWritter.Close()

	// Get ssh session
	ssh, err := core.NewSSHClient(host, sshUser, sshKey, []string{})
	if err != nil {
		log.Fatalln("ERR - unable to NewSSHClient -", err)
	}

	sshSession, err := ssh.GetSession()
	if err != nil {
		log.Fatalln("ERR - unable to GetSession -", err)
	}

	// stream is send to stdout
	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		log.Fatalln("ERR - unable to stdoutPipe -", err)
	}

	go io.Copy(fileWritter, stdout)

	sshSession.Stderr = os.Stdout

	if err := sshSession.Run(command); err != nil {
		sshSession.Close()
		log.Fatalln("ERR - unable to run", command, ". ", err)
	}
	sshSession.Close()

	// remove old backup
	if keep != "" {
		log.Println("removing old backups...")
		duration, err := time.ParseDuration(keep)
		if err != nil {
			log.Fatalln("ERR - ", keep, " is not a valid duration.")
		}
		files, err := ioutil.ReadDir(backupDir)
		if err != nil {
			log.Fatalln("ERR - unable to scan dir ", backupDir, ". ", err)
		}
		for _, file := range files {
			if file.ModTime().Before(time.Now().Add(-duration)) {
				os.RemoveAll(path.Join(backupDir, file.Name()))
			}
		}
	}
	log.Println("backup done.")
}
