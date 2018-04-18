//go:generate goversioninfo -icon=icon.ico
package main

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/skratchdot/open-golang/open"
	"gopkg.in/yaml.v2"
)

type metadata struct {
	Version string `yaml:"version"`
	Author  string `yaml:"author"`
	Index   string `yaml:"index"`
}

func main() {
	log.SetOutput(os.Stdout)
	if len(os.Args) < 2 {
		return
	}
	dir, err := ioutil.TempDir("", "dap-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	meta, err := openApplication(os.Args[1], dir)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Application:", getName(os.Args[1]))
	fmt.Println("Version:", meta.Version)
	fmt.Println("Author:", meta.Author)
	abort := make(chan bool)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(abort)
		cmd := exec.Command("docker-compose", "-f", filepath.Join(dir, "docker-compose.yml"), "up")
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "COMPOSE_CONVERT_WINDOWS_PATHS=1")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
	}()
	openIndex(abort, meta.Index)
	wg.Wait()
}

func getName(app string) string {
	return strings.TrimSuffix(filepath.Base(app), filepath.Ext(app))
}

func openApplication(app, dir string) (metadata, error) {
	r, err := os.Open(app)
	if err != nil {
		log.Fatal(err)
	}
	tr := tar.NewReader(r)
	var meta metadata
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return meta, nil
		}
		if err != nil {
			return meta, err
		}
		if hdr.Name == "docker-compose.yml" {
			f, err := os.Create(filepath.Join(dir, hdr.Name))
			if err != nil {
				return meta, err
			}
			if _, err = io.Copy(f, tr); err != nil {
				return meta, err
			}
		} else if hdr.Name == "docker-application.yml" {
			err = yaml.NewDecoder(tr).Decode(&meta)
			if err != nil {
				return meta, err
			}
		}
	}
	return meta, nil
}

func openIndex(abort chan bool, index string) {
	if index == "" {
		return
	}
	for {
		select {
		case <-abort:
			return
		default:
			address := strings.TrimPrefix(index, "http://")
			_, err := net.DialTimeout("tcp", address, time.Second)
			if err == nil {
				open.Start(index)
				return
			}
		}
	}
}
