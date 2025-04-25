package main

import (
   "flag"
   "fmt"
   "log"
   "encoding/json"
   "io/ioutil"
   "os"
   "path/filepath"
   "strings"
   "text/template"
   cp "github.com/otiai10/copy"
)

type SystemConfig struct {
   httpdChrootDir  string
   socketBaseDir   string      // where should socket files for dynamic sites be placed?
   staticBaseDir   string      // where should static files be placed?
   servicesDir     string      // where should services be installed?
   httpdConfigFilepath   string    // where should httpd config be placed?
   serviceFileExtension string
   httpdTemplateFilename   string
   serviceTemplateFilename string
   serviceFilePermissions int
}

type SiteInfo struct {
   Hostname        string          `json:"hostname"`
   IsDynamic       bool            `json:"is_dynamic"`
   ExecutablePath  string          `json:"exe_path"`
   SocketPath      string          `json:"socket_path"`
   SocketPathRel   string          `json:"socket_path_rel"`
   StaticFilesSourcePath string    `json:"static_files_src"`
   StaticFilesInstallPath string   `json:"static_files_dst"`
   StaticFilesInstallPathRel   string `json:"static_files_dst_rel"`
}

func readAllSites(sitesDir string, sysCfg SystemConfig) ([]SiteInfo, error) {
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
           continue
       }
       if !info.Mode().IsDir() {
           continue
       }

       thisSite := SiteInfo{Hostname: siteName}

       _, err = os.Stat(mainExe)
       if err == nil {
           thisSite.IsDynamic = true
           thisSite.ExecutablePath = mainExe
           thisSite.SocketPath = filepath.Join(sysCfg.socketBaseDir, fmt.Sprintf("%s.sock", siteName))
           thisSite.SocketPathRel, err = filepath.Rel(sysCfg.httpdChrootDir, thisSite.SocketPath)
           if err != nil {
               return nil, err
           }
       } else {
           thisSite.IsDynamic = false
           thisSite.StaticFilesSourcePath = buildDir
           thisSite.StaticFilesInstallPath = filepath.Join(sysCfg.staticBaseDir, siteName)
           thisSite.StaticFilesInstallPathRel, err = filepath.Rel(
               sysCfg.httpdChrootDir, thisSite.StaticFilesInstallPath)
           if err != nil {
               return nil, err
           }
       }

       // Add site info
       sitesInfo = append(sitesInfo, thisSite)
   }

   return sitesInfo, nil
}


func getServiceFilepath(hostname string, sysCfg SystemConfig) string {
    replacer := strings.NewReplacer(".", "_", "-", "_")
    filePath := filepath.Join(sysCfg.servicesDir,
        replacer.Replace(hostname) +
        sysCfg.serviceFileExtension)
    return filePath
}

func updateSystemConfigFiles(sitesInfo []SiteInfo, templateDir string, sysCfg SystemConfig) error {
   // Update httpd config file to route traffic to each site
   httpdTemplate, err := template.ParseFiles(filepath.Join(templateDir, sysCfg.httpdTemplateFilename))
   if err != nil {
       return err
   }
   f, err := os.Create(sysCfg.httpdConfigFilepath)
   if err != nil {
       return err
   }
   httpdTemplate.Execute(f, sitesInfo)

   // Create service file for each dynamic site and copy files for each static site
   serviceTemplate, err := template.ParseFiles(filepath.Join(templateDir,
       sysCfg.serviceTemplateFilename))
   if err != nil {
       return err
   }
   for _, site := range sitesInfo {
       if site.IsDynamic {
           filePath := getServiceFilepath(site.Hostname, sysCfg)
           filePermissions := os.FileMode(sysCfg.serviceFilePermissions)
            // Create the file with the specified permissions
           f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePermissions)
           if err != nil {
               return err
           }
           serviceTemplate.Execute(f, site)
       } else {
           opt := cp.Options {
               OnDirExists: func(src, dest string) cp.DirExistsAction {
                   return cp.Replace
               },
           }
           cp.Copy(site.StaticFilesSourcePath, site.StaticFilesInstallPath, opt)
       }
   }
   return nil
}


func sitesInfoFromJsonFile(filename string) ([]SiteInfo, error) {
   file, err := os.Open(filename)
   if err != nil {
       return nil, err
   }
   defer file.Close()

   var sitesInfo []SiteInfo
   byteValue, _ := ioutil.ReadAll(file)
   err = json.Unmarshal(byteValue, &sitesInfo);
   if err != nil {
       return nil, err
   }
   return sitesInfo, nil

}


func sitesInfoToJsonFile(filename string, sitesInfo []SiteInfo) error {
   sitesJson, err := json.MarshalIndent(sitesInfo, "", "    ")
   err = os.WriteFile(filename, sitesJson, 0644)
   return err
}


func removeSystemConfigFiles(sitesInfo []SiteInfo, sysCfg SystemConfig) error {
   // Remove httpd config file
   err := os.Remove(sysCfg.httpdConfigFilepath)
   if err != nil {
       log.Print(err)
   }

   // Remove service files
   for _, site := range sitesInfo {
       if site.IsDynamic {
           removeFilepath := getServiceFilepath(site.Hostname, sysCfg)
           err = os.Remove(removeFilepath)
           if err != nil {
               log.Print(err)
           }
       } else {
           err = os.RemoveAll(site.StaticFilesInstallPath)
           if err != nil {
               log.Print(err)
           }
       }
   }

   return nil
}


func main() {
   flag.Parse()

   ex, err := os.Executable()
   if err != nil {
       log.Fatal(err)
   }
   exPath := filepath.Dir(ex)

   sitesDir     := "/sites"
   templateDir  := filepath.Join(exPath, "templates")
   jsonFilename := filepath.Join(exPath, "sites.json")

   /*systemConfigUbuntuDev := SystemConfig {
       httpdChrootDir: "/",
       serviceTemplateFilename: "site.service",
       httpdTemplateFilename: "nginx.conf",
       socketBaseDir: "/tmp/sitemgr/sock",
       staticBaseDir: "/tmp/sitemgr/www",
       servicesDir:   "/tmp/sitemgr/service",
       httpdConfigFilepath: "/tmp/sitemgr/nginx.conf",
       serviceFileExtension: ".service",
       serviceFilePermissions: 0660,
   }*/

   systemConfigOpenBsd := SystemConfig {
       httpdChrootDir: "/var/www",
       serviceTemplateFilename: "site.rc",
       httpdTemplateFilename: "httpd.conf",
       socketBaseDir: "/var/www/run",
       staticBaseDir: "/var/www/htdocs",
       servicesDir:   "/etc/rc.d/",
       httpdConfigFilepath: "/etc/httpd.conf",
       serviceFileExtension: "",
       serviceFilePermissions: 0555,
   }

   sysCfg := systemConfigOpenBsd

   // Infer site info from directory structure
   sitesInfo, err := readAllSites(sitesDir, sysCfg)
   if err != nil {
       log.Fatal(err)
   }

   // Remove previously-deployed managed services and static files
   oldSitesInfo, err := sitesInfoFromJsonFile(jsonFilename)
   if err != nil {
       log.Printf("invalid or missing file %s; skipping removal of old files", jsonFilename)
   } else {
       err = removeSystemConfigFiles(oldSitesInfo, sysCfg)
       if err != nil {
           log.Fatal(err)
       }
   }

   // Install new updated services, config files, and static files
   err = updateSystemConfigFiles(sitesInfo, templateDir, sysCfg)
   if err != nil {
       log.Fatal(err)
   }

   // Save current sitesInfo to json so it can be used to remove stale files
   // on next run
   err = sitesInfoToJsonFile(jsonFilename, sitesInfo)
   if err != nil {
       log.Fatal(err)
   }
}
