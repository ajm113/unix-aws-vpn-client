package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	OpenVPNSourceFolderName = "openvpn-2.5.1"
	OpenVPNTarName          = "openvpn-2.5.1.tar.xz"
	OpenVPNSource           = "https://swupdate.openvpn.org/community/releases/openvpn-2.5.1.tar.xz"
)

var (
	OpenVPNPatchScript      = path.Join("scripts", "openvpn-v2.5.1-aws.patch")
	OpenVPNConfigureOptions = []string{
		"--disable-debug",
		"--disable-dependency-tracking",
		"--disable-silent-rules",
		"--with-crypto-library=openssl",
		// This maybe a Apple specific thing you want to enable.
		// Tested this on Arch and Debian with no issues...
		// "--enable-pkcs11",
	}
)

// setupAction Compiles and builds patched version of openvpn and verify we have everything we need before doing so.
// TODO: Download and patch and compile via Golang?
func setupAction(c *cli.Context) error {
	patchFile := c.String("patch")
	outputDir := c.String("out")
	sourceDir := c.String("source")

	// TODO: Remove me when Windows has been fully supported and tested.
	if runtime.GOOS == "windows" {
		log.Fatal().Msg("Detected windows environment! This operation is not properly developed to execute for Windows. Please manually build openvpn using the provided ruby script." + errorSuffix)
	}

	// Make sure all required commands are installed on the system.
	// TODO: Add more commands to check?
	var quit bool

	if !commandExists("make") {
		log.Error().Msg("Make not found! Please install build-essentials or development tools to continue." + errorSuffix)
		quit = true
	}

	if !commandExists("patch") {
		log.Error().Msg("Make not found! Please install build-essentials or development tools to continue." + errorSuffix)
		quit = true
	}

	if !commandExists("wget") {
		log.Error().Msg("Make not found! Please install build-essentials or development tools to continue. " + errorSuffix)
		quit = true
	}

	if !commandExists("tar") {
		log.Error().Msg("Make not found! Please install build-essentials or development tools to continue. " + errorSuffix)
		quit = true
	}

	if quit {
		return fmt.Errorf("one or more commands not found")
	}

	if patchFile == "" {
		patchFile = OpenVPNPatchScript
	}

	if !fileExists(patchFile) {
		log.Error().Msgf("Patch file '%s' not found! Please use -p to define a patch file! "+errorSuffix, patchFile)
	}

	if sourceDir == "" {
		log.Info().Msgf("Downloading and extracing %s...", OpenVPNSourceFolderName)
		tempDir := os.TempDir()
		err := downloadOpenVPN(tempDir)

		if err != nil {
			return fmt.Errorf("failed downloading OpenVPN")
		}

		tarFilename := path.Join(tempDir, OpenVPNTarName)
		err = extractTarFile(tarFilename, tempDir)

		if err != nil {
			return fmt.Errorf("failed extracting OpenVPN")
		}

		sourceDir = path.Join(tempDir, OpenVPNSourceFolderName)
	}

	log.Info().Msgf("Applying patch %s...", sourceDir)
	err := patchOpenVPN(sourceDir, patchFile)
	if err != nil {
		return fmt.Errorf("failed patching OpenVPN source code")
	}

	log.Info().Msgf("Compiling OpenVPN...")
	err = compileOpenVPN(sourceDir, outputDir)
	if err != nil {
		return fmt.Errorf("failed patching OpenVPN source code")
	}

	return nil
}

func downloadOpenVPN(dir string) error {
	log.Debug().Str("url", OpenVPNSource).Str("dir", dir).Msg("Downloading OpenVPN...")

	out, err := exec.Command("wget", "-P", dir, "-O", path.Join(dir, OpenVPNTarName), OpenVPNSource).CombinedOutput()

	if err != nil {
		log.Error().Err(err).Bytes("out", out).Msg("Failed downloading OpenVPN " + errorSuffix)
		return err
	}

	return nil
}

func extractTarFile(filename, dir string) error {
	log.Debug().Str("filename", filename).Str("dir", dir).Msg("Extracting tar...")
	out, err := exec.Command("tar", "-xvf", filename, "-C", dir).CombinedOutput()

	if err != nil {
		log.Error().Err(err).Bytes("out", out).Msg("Failed extracting OpenVPN " + errorSuffix)
		return err
	}

	return nil
}

func patchOpenVPN(source, patchFilename string) error {
	log.Debug().Msgf("Running 'patch -p 1 -d %s < %s'", source, patchFilename)
	cmd := exec.Command("/bin/bash", "-c", "/usr/bin/patch", "-p", "1", "-d", source, "<", patchFilename)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Error().Err(err).Bytes("out", out).Msg("Failed patching OpenVPN source code! " + errorSuffix)
		return err
	}

	return nil
}

func compileOpenVPN(source, outputDir string) error {
	log.Debug().Strs("args", OpenVPNConfigureOptions).Msg("running configure...")
	configureCmd := exec.Command("./configure", OpenVPNConfigureOptions...)
	configureCmd.Stdout = os.Stdout
	configureCmd.Stderr = os.Stderr
	configureCmd.Stdin = os.Stdin
	configureCmd.Dir = source

	err := configureCmd.Start()
	if err != nil {
		log.Error().Err(err).Msg("Failed compiling OpenVPN! " + errorSuffix)
		return err
	}

	err = configureCmd.Wait()

	if err != nil {
		log.Error().Err(err).Msg("Failed compiling OpenVPN! " + errorSuffix)
		return err
	}

	log.Debug().Msg("running make...")
	makeCmd := exec.Command("make")
	makeCmd.Stdout = os.Stdout
	makeCmd.Stderr = os.Stderr
	makeCmd.Stdin = os.Stdin
	makeCmd.Dir = path.Join(source, "src")

	err = makeCmd.Start()
	if err != nil {
		log.Error().Err(err).Msg("Failed compiling OpenVPN! " + errorSuffix)
		return err
	}

	err = makeCmd.Wait()

	if err != nil {
		log.Error().Err(err).Msg("Failed compiling OpenVPN! " + errorSuffix)
		return err
	}

	binaryFilename := path.Join(source, "src", "openvpn", "openvpn")
	distFilename := path.Join(outputDir, "openvpn_aws")
	if !fileExists(binaryFilename) {
		log.Error().Err(err).Msg("Failed compiling OpenVPN! " + errorSuffix)
		return fmt.Errorf("binary '%s' failed to compile", binaryFilename)
	}

	err = copyFile(binaryFilename, path.Join(outputDir, distFilename))
	if err != nil {
		log.Error().Err(err).Str("bin", binaryFilename).Str("dist", distFilename).Msg("Failed copying dist binary " + errorSuffix)
		return fmt.Errorf("failed copy")
	}

	log.Info().Msgf("Finished compiling binary: %s", distFilename)
	log.Info().Msg("Make sure to move this binary to a safe spot and make sure to setup your awsvpnclient.yml vpn.openvpn field to this executable!")

	return nil
}
