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

// Create a new SFTP connection by given parameters
func CreateNewConnection(host, user, password string, port int) (client *SftpClient, err error) {
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
	if err != nil {
		log.Println("connection Err => %s", err)
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
func (sc *SftpClient) Put(dataLocalFile, remoteFileName string) (err error) {

	// // Make remote directories recursion
	parent := filepath.Dir(remoteFileName)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		log.Println("put path =>", path)
		sc.Mkdir(path)
	}

	createEmptySftpFileFromRemoteFile, createEmptySftpFileErr := sc.Create("sftpuser/" + remoteFileName)
	if createEmptySftpFileErr != nil {
		return
	}
	defer createEmptySftpFileFromRemoteFile.Close()

	openRemoteFileForCopy, openRemoteFileErr := os.Open(dataLocalFile)
	if openRemoteFileErr != nil {
		log.Println("openRemoteFileErr =>", openRemoteFileErr)
		return
	}
	defer openRemoteFileForCopy.Close()

	copyOpenRemoteFileToEmptyFile, err := io.Copy(createEmptySftpFileFromRemoteFile, openRemoteFileForCopy)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d copyOpenRemoteFileToEmptyFile copied\n", copyOpenRemoteFileToEmptyFile)

	defer os.Remove(dataLocalFile) // deleteFile uploaded on server before transfer to sftp
	return
}

// Download file from sftp server
func (sc *SftpClient) Get(remoteFile, localFile string) (err error) {
	openLocalFile, err := sc.Open("sftpuser/" + remoteFile)
	if err != nil {
		log.Println("openLocalFile err => ", err)
		return
	}
	defer openLocalFile.Close()
	os.Chdir("./downloadFolder")
	createLocalFileFromRemoteFile, err := os.Create(localFile)
	if err != nil {
		log.Println("createLocalFileFromRemoteFile err => ", err)
		return
	}
	defer createLocalFileFromRemoteFile.Close()

	_, err = io.Copy(createLocalFileFromRemoteFile, openLocalFile)

	if err != nil {
		log.Println("Copy err => ", err)
		return
	}

	return
}

// SSH Key-strings
func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {

	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		keyString(k)
		return nil
	}
}

func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal())
}
