package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func copyFile(source, dest string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest) // creates if file doesn't exist
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile) // check first var for number of bytes copied
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func isRoot() bool {
	return syscall.Geteuid() == 0
}

func openDefaultBrowser(defaultUser, url string) (err error) {
	switch runtime.GOOS {
	case "linux":
		err = commandAndStartAsNonRoot(defaultUser, "xdg-open", url)
	case "darwin":
		err = commandAndStartAsNonRoot(defaultUser, "open", url)
	default:
		err = fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}

	return
}

func commandExists(command string) bool {
	cmd := exec.Command("/bin/sh", "-c", "command -v "+command)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}

	return false
}

func getHomeDirConfigPath() (foldername string, err error) {
	home, err := os.UserHomeDir()

	if err != nil {
		return
	}

	configFolder := ".config"

	if runtime.GOOS == "windows" {
		configFolder = "AppData\\Local"
	}

	foldername = path.Join(home, configFolder, defaultConfigDirectoryName)

	return
}

func searchConfigFilename() (string, error) {

	wd, wdErr := os.Getwd()

	if wdErr != nil {
		return "", wdErr
	}

	configAtwd := path.Join(wd, defaultConfigFilename)

	home, _ := getHomeDirConfigPath()
	configAtHome := path.Join(home, defaultConfigFilename)

	if fileExists(configAtwd) {
		return configAtwd, nil
	} else if fileExists(configAtHome) {
		return configAtHome, nil
	}

	return "", os.ErrNotExist
}

func lookupIP(hostname string) (ip string, err error) {
	ips, err := net.LookupIP(hostname)

	if err != nil || len(ips) == 0 {
		return
	}

	for _, ipv4 := range ips {
		ip = ipv4.String()
		break
	}

	return
}

func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func extractSIDFromOpenVPN(output string) (SID string, err error) {
	tokens := strings.Split(output, ":")

	for _, t := range tokens {
		if strings.HasPrefix(t, "instance-") {
			SID = t
			break
		}
	}

	if SID == "" {
		err = fmt.Errorf("sid not found")
	}

	return
}

// shorthand for exec.Command(command, args...).Start() except it does SysProcAttr and Env injection
// to ensure web browsers can safely startup.
//
// - defaultUser Should only be filled if for some reason the SUDO_USER env variable wont exist.
// - command Binary we want to execute.
func commandAndStartAsNonRoot(defaultUser string, command string, args ...string) error {

	// If we aren't running as root. We just run exec.Command normally.
	if !isRoot() {
		return exec.Command(command, args...).Start()
	}

	userName := os.Getenv("SUDO_USER")
	if userName == "" {
		userName = defaultUser
	}

	// Get the user information for the non-root user (e.g., the user running the app initially)
	nonRootUser, err := user.Lookup(userName) // Replace "your_username" with the actual user
	if err != nil {
		return fmt.Errorf("failed to lookup user: %w", err)
	}

	// Parse the UID and GID of the non-root user
	uid, err := strconv.Atoi(nonRootUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to parse UID: %w", err)
	}
	gid, err := strconv.Atoi(nonRootUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to parse GID: %w", err)
	}

	// Prepare the command to execute
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	cmd.Env = append(os.Environ(),
		"HOME="+nonRootUser.HomeDir,
		"USER="+nonRootUser.Username,
		"LOGNAME="+nonRootUser.Username,
		"DISPLAY="+os.Getenv("DISPLAY"),
		"XDG_RUNTIME_DIR=/run/user/"+nonRootUser.Uid,
	)

	// Start the command as the non-root user
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	return nil
}
