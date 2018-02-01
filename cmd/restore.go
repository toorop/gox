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
	"log"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/toorop/gox/core"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore a backup on a remote host",
	Long: `Restore a backup on a remote host

	gox --restore --config path/to/config.yaml --from backupFolder

Example:
	gox --restore --config server1.yaml --from 2018-02-01--15-16-55
	`,
	Run: restore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)
	restoreCmd.Flags().String("from", "", "full path of backup.xbstream")
}

func restore(cmd *cobra.Command, args []string) {

	sshUser := viper.GetString("ssh.user")
	sshKey := viper.GetString("ssh.key")

	// host
	host := viper.GetString("host")
	if host == "" {
		log.Fatalln("ERR - 'host' not found in config")
	}

	// target
	backupDir := viper.GetString("backup-dir")
	if backupDir == "" {
		log.Fatalln("ERR - 'backup-dir' not found in config")
	}

	// from
	from := cmd.Flags().Lookup("from").Value.String()
	if from == "" {
		log.Fatalln("ERR - 'from' not found from command line")
	}
	from = path.Join(backupDir, from)

	fmt.Printf("Restoring %s to %s ? (y/N): ", from, host)
	var resp string
	fmt.Scanf("%s", &resp)
	if resp != "y" {
		os.Exit(0)
	}

	ssh, err := core.NewSSHClient(host, sshUser, sshKey, []string{"stdout", "stderr"})
	if err != nil {
		log.Fatalln("ERR - unable to get ssh client", host, ".", err)
	}

	// scp
	tempDir := path.Join("/tmp", time.Now().Format("2006-01-02--15-04-05"))
	log.Println(fmt.Sprintf("Transfering backup to %s:%s...", host, path.Join(tempDir, "backup.xbstream")))

	// tempdir
	ssh.RunOrDie(fmt.Sprintf("mkdir -p %s", tempDir))
	defer func() {
		// Remove tempdir
		ssh.RunOrDie(fmt.Sprintf("rm -rf %s", tempDir))
	}()

	// scp
	if err := ssh.CopyFile(path.Join(from, "backup.xbstream"), path.Join(tempDir, "backup.xbstream"), "0644"); err != nil {
		log.Fatalln("ERR - unable to transfert backup -", err)
	}

	log.Println("Decompressing backup...")
	// Destream
	ssh.RunOrDie(fmt.Sprintf("xbstream -C %s -x < %s/backup.xbstream", tempDir, tempDir))

	// Decompress
	ssh.RunOrDie(fmt.Sprintf("xtrabackup --remove-original --decompress --target-dir=\"%s/\"", tempDir))

	// Prepare
	log.Println("Preparing backup...")
	ssh.RunOrDie(fmt.Sprintf("xtrabackup --prepare --target-dir=%s", tempDir))

	// Shutdown MySQL
	log.Println("Shuting down MySQL")
	ssh.RunOrDie("service mysql stop")

	// mv /var/lib/mysql (just in case...)
	log.Println("Moving /var/lib/mysql to /var/lib/mysql.bck")
	ssh.RunOrDie("rm -rf /var/lib/mysql.bck; mv -f /var/lib/mysql /var/lib/mysql.bck")

	// Mkdir /var/lib/mysql
	ssh.RunOrDie("mkdir /var/lib/mysql")

	// Restore
	// xtrabackup --copy-back --target-dir=/data/backups/
	log.Println("Restoring backup...")
	ssh.RunOrDie(fmt.Sprintf("xtrabackup --copy-back --target-dir=%s", tempDir))

	// chown -R mysql:mysql /var/lib/mysql/
	ssh.RunOrDie("chown -R mysql:mysql /var/lib/mysql/")

	// Start Mysql
	// if galera node
	command := "service mysql start"
	if viper.GetBool("galera") {
		out, err := ssh.GetOutput(fmt.Sprintf("cat %s", path.Join(tempDir, "xtrabackup_galera_info")))
		if err != nil {
			log.Fatalln("ERR - unable to get xtrabackup_galera_info.", err)
		}
		command = fmt.Sprintf("%s --wsrep_start_position=\"%s\"", command, string(out))
	}
	log.Println("Starting MySQL...")

	ssh.RunOrDie(command)

	// That's it !
	log.Println("Backup restored !")
}
