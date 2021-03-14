package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"github.com/secsy/goftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	indexesDir = "indexes"
)

const projectDir = "F:\\Downloads\\OpenServer\\domains\\booking_local"

//const projectDir = "F:\\Downloads\\OpenServer\\domains\\symf5api"
//const projectDir = "C:\\Users\\user\\Documents\\projects\\test_go\\indexes"

//const projectDirFtp = "/FreeAgent_Drive/projects/booking/"
const projectDirFtp = "/FreeAgent Drive/projects/indexes"

var projectDirs = [...]string{
	//"F:\\Downloads\\OpenServer\\domains\\booking_local",
	//"F:\\Downloads\\OpenServer\\domains\\symf5api",
	//"C:\\Users\\user\\Documents\\projects\\test_go\\indexes",
	"/home/owner/Documents/projects/go/cloud-programming/indexes",
}

func main() {
	createDir(indexesDir)

	go exitProgram()

	//client := createSftpClient()
	//if client != nil {
	//	defer client.Close()
	//
	//	rootDir, err := client.Getwd()
	//	check(err)
	//	if err == nil {
	//		log.Println(rootDir)
	//	}
	//
	//	cwd, err := client.ReadDir(rootDir)
	//	for _, val := range cwd {
	//		log.Println(val.Name())
	//	}
	//
	//	cwd2, err := client.Stat(rootDir + projectDirFtp)
	//	check(err)
	//	if err == nil {
	//		log.Println(cwd2.Name(), cwd2.IsDir())
	//	}
	//}

	for {
		for indexDir, dirRoot := range projectDirs {
			log.Println(indexDir+1, dirRoot)

			start := time.Now()

			files, err := getFiles(dirRoot, projectDirFtp)
			//files, err := FilePathWalkDir(dirRoot)

			check(err)

			log.Printf("Count files: %d, processing time: %f", len(files), time.Now().Sub(start).Seconds())
			log.Println("Files: ", files)
			//log.Printf("Processing time: %f", time.Now().Sub(start).Seconds())
		}

		time.Sleep(10 * time.Second)
	}
}

func exitProgram() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Exiting")
	os.Exit(0)
}

func check(e error) {
	if e != nil {
		log.Println(e)
	}
}

func createDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.Mkdir(dirName, os.ModePerm)
		check(err)
	}
}

func getFiles(root, rootFtp string) ([]string, error) {
	var outputFiles []string
	c := make(chan string)
	cQuit := make(chan byte)

	client := createSftpClient()

	if client == nil {
		log.Println("FTP client is nil")
		return nil, nil
	}

	defer client.Close()

	cwd, err := client.Getwd()
	if err != nil {
		log.Println(err)
		return nil, nil
	}

	rootFtp = cwd + rootFtp

	go getFile(root, rootFtp, client, "/", nil, &c, &cQuit)

	for {
		select {
		case fileName := <-c:
			outputFiles = append(outputFiles, fileName)
		case <-cQuit:
			return outputFiles, nil
		}
	}

}

func getFile(root, rootFtp string, ftpClient *sftp.Client, path string, wg *sync.WaitGroup, c *chan string, cQuit *chan byte) {
	var localWg sync.WaitGroup

	if wg != nil {
		defer wg.Done()
	}

	files, err := ioutil.ReadDir(root + path)

	//f, err := os.Open(root)
	//if err != nil {
	//	log.Println(err)
	//	return
	//}
	//files, err := f.Readdir(-1)
	//_ = f.Close()

	if err != nil {
		log.Println(err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			localWg.Add(1)
			go getFile(root, rootFtp, ftpClient, fmt.Sprintf("%s%s/", path, file.Name()), &localWg, c, nil)
		} else {
			//if time.Now().Sub(file.ModTime()).Seconds() < 20 {
			//*c <- path + file.Name()
			//}

			fileInfo, err := ftpClient.Stat(rootFtp + path + file.Name())
			var dstFile *sftp.File

			if os.IsNotExist(err) {
				dstFile, err = ftpClient.Create(rootFtp + path + file.Name())
				if err != nil {
					log.Println(err)
					continue
				}
			}

			if dstFile != nil || fileInfo != nil {
				if file.ModTime().UTC().After(fileInfo.ModTime().UTC().Add(10 * time.Second)) {
					srcFile, err := os.Open(root + path + file.Name())
					check(err)

					if srcFile != nil {
						*c <- path + file.Name()

						_, err = io.Copy(dstFile, srcFile)
						check(err)

						_ = srcFile.Close()

						//err = client.Delete(rootFtp + path + file.Name())
						//check(err)
						//err = client.Store(rootFtp+path+file.Name(), storedFile)
						//check(err)
						//_ = storedFile.Close()
					}
				}
			} else {
				check(err)
			}

			_ = dstFile.Close()
		}
	}

	localWg.Wait()

	if cQuit != nil {
		*cQuit <- 1
	}
}

func FilePathWalkDir(root string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			//files = append(files, path)
			if time.Now().Sub(info.ModTime()).Seconds() < 20 {
				files = append(files, info.Name())
			}
			//log.Println(info.Name(), info.ModTime())
		}
		return nil
	})

	return files, err
}

func createFtpClient() *goftp.Client {
	config := goftp.Config{
		User:     "admin",
		Password: "1234512345",
		//Logger:             os.Stderr,
		//ConnectionsPerHost: 10,
		//Timeout:            10 * time.Second,
	}

	client, err := goftp.DialConfig(config, "192.168.1.1")
	if err != nil {
		log.Println(err)
		return nil
	}

	return client
}

func createSftpClient() *sftp.Client {
	addr := "192.168.1.1:22"
	user := "admin"
	pass := "1234512345"

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		log.Println(err)
		return nil
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Println(err)
		return nil
	}

	return client
}
