package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

const (
	sitesDir      = "sites"
	socketBaseDir = "/var/www/run"
	staticBaseDir = "/var/www/htdoc"
	httpdTemplateFilename = "httpd-template.conf"
	httpdConfigFilepath = "/etc/httpd.conf"
)

type SiteInfo struct {
	Hostname        string
	IsDynamic       bool
	ExecutablePath  string
	SocketPath      string
	StaticFilesPath string
}

func readAllSites(sitesDir string) ([]SiteInfo, error) {
	// Read the provided sites directory and create a table of site information
	// site hostname, type (static vs. dynamic), allocated port if dynamic
	sitesInfo := []SiteInfo{}

	files, err := os.ReadDir(sitesDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		siteName := file.Name()
		buildDir := filepath.Join(sitesDir, file.Name(), "build")
		mainExe := filepath.Join(buildDir, "main")

		// Skip non-directories or files where stat fails
		info, err := os.Stat(buildDir)
		if err != nil {
			log.Printf("Stat error: %s %s", file, err)
			continue
		}
		if !info.Mode().IsDir() {
			log.Printf("%s is not a directory", file)
			continue
		}

		thisSite := SiteInfo{Hostname: siteName}

		_, err = os.Stat(mainExe)
		if err == nil {
			thisSite.IsDynamic = true
			thisSite.ExecutablePath = mainExe
			thisSite.SocketPath = filepath.Join(socketBaseDir, fmt.Sprintf("%s.sock", siteName))
		} else {
			thisSite.IsDynamic = false
			log.Printf("Static site detected: %s", file)
			log.Print(err)
			thisSite.StaticFilesPath = filepath.Join(staticBaseDir, siteName)
		}

		// Add site info
		sitesInfo = append(sitesInfo, thisSite)
	}

	return sitesInfo, nil
}

func readAndPrintSitesInfo() {
	sitesInfo, err := readAllSites(sitesDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(sitesInfo)


	// TODO: create or update service files for each site
	

	// Update httpd config file to route traffic to each site
	httpdTemplate, err := template.ParseFiles(httpdTemplateFilename)
	if err != nil {
		log.Fatal(err)
	}
	f, err := os.Create(httpdConfigFilepath)
	if err != nil {
		log.Fatal(err)
	}
	httpdTemplate.Execute(f, sitesInfo)
}

func main() {
	flag.Parse()
	readAndPrintSitesInfo()
}

