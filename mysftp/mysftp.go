package mysftp

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpClient struct {
	host, user, password string
	port                 int
	*sftp.Client
}

const sshTurstedKey = "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBN8nr6yUiSDLaAjbgtdBjJtn6xvnDbeAU7AbW76Li0Ht29Tc4tWWJZ8puOpPwu2/YMZCRn15OVQlz3XtH6JqClw="

// Create a new SFTP connection by given parameters
func NewConn(host, user, password string, port int) (client *SftpClient, err error) {
	switch {
	case `` == strings.TrimSpace(host),
		`` == strings.TrimSpace(user),
		`` == strings.TrimSpace(password),
		0 >= port || port > 65535:
		return nil, errors.New("Invalid parameters")
	}

	client = &SftpClient{
		host:     host,
		user:     user,
		password: password,
		port:     port,
	}
	log.Println(client)
	if err = client.connect(); nil != err {
		return nil, err
	}
	return client, nil
}

func (sc *SftpClient) connect() (err error) {

	config := &ssh.ClientConfig{
		User:            sc.user,
		Auth:            []ssh.AuthMethod{ssh.Password(sc.password)},
		Timeout:         30 * time.Second,
		HostKeyCallback: trustedHostKeyCallback(""),
	}

	// connet to ssh
	addr := fmt.Sprintf("%s:%d", sc.host, sc.port)
	conn, err := ssh.Dial("tcp", addr, config)
	log.Println("connection =>", conn, err)
	if err != nil {
		return err
	}

	// create sftp client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	sc.Client = client

	return nil
}

// Upload file to sftp server
func (sc *SftpClient) Put(localFile, remoteFile string) (err error) {
	nameDir, err := os.Getwd()

	log.Println("put nameDir + err =>", nameDir, err)
	log.Println("put localFile =>", localFile)
	log.Println("put remoteFile =>", remoteFile)

	// Make remote directories recursion
	parent := filepath.Dir(remoteFile)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		log.Println("put path =>", path)
		sc.Mkdir(path)
	}

	dstFile, err := sc.Create("sftpuser/" + remoteFile)
	log.Println("create file + err =>", dstFile, err)
	if err != nil {
		return
	}
	defer dstFile.Close()

	srcFile, err := os.Open(localFile)
	log.Println("srcFile +err =>", srcFile, err)
	if err != nil {
		return
	}
	// defer srcFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d bytes copied\n", bytes)
	defer os.Remove(localFile) // clean up
	return
}

// Download file from sftp server
func (sc *SftpClient) Get(remoteFile, localFile string) (err error) {
	srcFile, err := sc.Open(remoteFile)
	if err != nil {
		return
	}
	defer srcFile.Close()

	dstFile, err := os.Create(localFile)

	if err != nil {
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return
}

// SSH Key-strings
func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {

	// if trustedKey == "" {
	// 	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
	// 		log.Println("k =>", k)
	// 		log.Printf("WARNING: SSH-key verification is *NOT* in effect: to fix, add this trustedKey: %q", keyString(k))
	// 		return nil
	// 	}
	// }

	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		// ks :=
		keyString(k)
		// if trustedKey != ks {
		// 	return fmt.Errorf("SSH-key verification: expected %q but got %q", trustedKey, ks)
		// }

		return nil
	}
}

func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal())
}
