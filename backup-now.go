package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	configFile = "config.txt"
	camputBin  = "camput.exe"
	//	camputBin  = "camput"
	camputConfig = "client-config.json"
	blobRootLog  = "log.txt"
)

var user, password, server string

type simpleConfig struct {
	Auth   string `json:"auth"`
	Server string `json:"server"`
}

func genCamputConfig() {
	if user == "" || password == "" || server == "" {
		log.Fatalf("All of USER, PASSWORD, and SERVER must be specified in %v", configFile)
	}
	// we always overwrite the camput config, in case the parameters
	// changed in configFile.
	f, err := os.Create(camputConfig)
	if err != nil {
		log.Fatalf("Failed to create camput config file %v: %v", camputConfig, err)
	}
	defer f.Close()
	conf := simpleConfig{
		Auth:   fmt.Sprintf("userpass:%v:%v", user, password),
		Server: server,
	}
	jsonEncoder := json.NewEncoder(f)
	if err := jsonEncoder.Encode(conf); err != nil {
		log.Fatalf("Failed to create camput configuration: %v", err)
	}
}

var blobRefRgx = regexp.MustCompile(`^sha1+-[a-f0-9]+$`)

func dirBlobRef(b *bytes.Buffer) string {
	br := strings.TrimSuffix(b.String(), "\n")
	if !blobRefRgx.MatchString(br) {
		log.Fatalf("Failed to get blobref from camput's output, got %v", br)
	}
	return br
}

func main() {
	// Sanity checks
	if _, err := os.Stat(configFile); err != nil {
		log.Fatalf("Please provide a \"%v\" file in the same directory, containing the paths to the directories to backup (one per line).", configFile)
	}
	if _, err := os.Stat(camputBin); err != nil {
		log.Fatalf("Please place %v in the same directory as this program.", camputBin)
	}

	// Scan the config file
	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open %v: %v", configFile, err)
	}
	defer f.Close()
	var dirs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := strings.ToLower(strings.Replace(scanner.Text(), " ", "", -1))
		switch {
		case strings.HasPrefix(l, "user="):
			user = strings.TrimPrefix(l, "user=")
		case strings.HasPrefix(l, "password="):
			password = strings.TrimPrefix(l, "password=")
		case strings.HasPrefix(l, "server="):
			server = strings.TrimPrefix(l, "server=")
		default:
			dirs = append(dirs, scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read %v: %v", configFile, err)
	}

	// Create the camput config file
	genCamputConfig()

	// Setup the output
	var output bytes.Buffer
	lw, err := os.OpenFile(blobRootLog, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		log.Fatalf("Could not open %v: %v", blobRootLog, err)
	}
	defer lw.Close()
	logger := log.New(lw, "", log.LstdFlags)

	// Setup the command(s)
	configDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	env := os.Environ()
	env = append(env, "CAMLI_CONFIG_DIR="+configDir)
	commonArgs := []string{"file"}

	for _, dir := range dirs {
		args := append(commonArgs, dir)
		cmd := exec.Command(camputBin, args...)
		cmd.Env = env
		cmd.Stdout = &output
		cmd.Stderr = os.Stderr
		log.Printf("Now running %v %v", camputBin, args)
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to upload dir %v: %v", dir, err)
		}
		// We rely on the camput output to be the bloref for
		// the root node of the dir. Sane enough?
		br := dirBlobRef(&output)
		logger.Printf("%v: %v", dir, br)
		output.Reset()
	}
	log.Print("Upload successful")
}
