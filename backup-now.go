package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
)

const (
	configFile = "config.txt"
	camputBin  = "camput.exe"
)

func main() {
	if _, err := os.Stat(configFile); err != nil {
		log.Fatalf("Please provide a \"%v\" file in the same directory, containing the paths to the directories to backup (one per line).", configFile)
	}
	if _, err := os.Stat(camputBin); err != nil {
		log.Fatalf("Please place %v in the same directory as this program.", camputBin)
	}

	f, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open %v: %v", configFile, err)
	}
	defer f.Close()
	var dirs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		dirs = append(dirs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read %v: %v", configFile, err)
	}

	commonArgs := []string{"file", "-permanode"}
	for _, dir := range dirs {
		args := append(commonArgs, dir)
		cmd := exec.Command(camputBin, args...)
		log.Printf("%v %v", camputBin, args)
		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to upload dir %v: %v", dir, err)
		}
	}
	log.Print("Upload sucessful")
}
