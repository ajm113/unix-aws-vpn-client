package main

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"

	"embed"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"mvdan.cc/xurls/v2"
)

type (
	serveHandle struct {
		Config                  *config
		OpenVPNConnectionConfig *openVPNConfig
		TempDir                 string

		SAMLResponse chan string
		ServiceIPv4  string
		ServiceHost  string
	}
)

//go:embed html/index.html
var welcomeHtmlFile embed.FS

//go:embed html/error.html
var errorHtmlFile embed.FS

func serveAction(c *cli.Context) error {
	openVPNConfig := c.String("config")
	tmpOpenVPNConfigDir := c.String("configTmpDir")
	awsClientConfigFilename, err := searchConfigFilename()

	if errors.Is(os.ErrNotExist, err) {
		log.Fatal().Msg("failed loading " + appName + " config from working directory and user config folder! " + errorSuffix)
	} else if err != nil {
		log.Fatal().Err(err).Msg("unexpected error loading " + appName + " config! " + errorSuffix)
	}

	awsclientConfig, err := loadConfig(awsClientConfigFilename)

	if err != nil {
		log.Fatal().Str("config", awsClientConfigFilename).Err(err).Msg("unexpected error loading " + appName + " config! " + errorSuffix)
	}

	if !awsclientConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().
		Str("config", openVPNConfig).
		Str("configOutDir", tmpOpenVPNConfigDir).
		Msg("Parsing openvpn config and saving formatted version for openvpn")

	connectionConfig, err := parseAndFormatOpenVPNConfig(openVPNConfig, tmpOpenVPNConfigDir)

	if err != nil {
		log.Fatal().
			Str("config", openVPNConfig).
			Str("configOut", tmpOpenVPNConfigDir).
			Err(err).
			Msg("Failed parsing or saving formatted version openvpn config! " + errorSuffix)
	}

	if connectionConfig.Formatted {
		log.Info().Str("config", connectionConfig.Filename).Msg("Parsed and formatted openvpn configuration.")
	} else {
		log.Info().Msg("Parsed openvpn configuration.")
	}

	handle := &serveHandle{
		Config:                  awsclientConfig,
		OpenVPNConnectionConfig: connectionConfig,
		SAMLResponse:            make(chan string),
		TempDir:                 tmpOpenVPNConfigDir,
	}

	log.Info().Msgf("Starting HTTP server at: %s", handle.Config.Server.Addr)
	go startSAMLServer(handle)

	startOpenVPNConnection(handle)

	return nil
}

func startOpenVPNConnection(handle *serveHandle) {
	connectionHostnameToken, err := generateRandomToken(12)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed generating random token for remote hostname! " + errorSuffix)
	}

	handle.ServiceHost = connectionHostnameToken + "." + handle.OpenVPNConnectionConfig.Host

	handle.ServiceIPv4, err = lookupIP(handle.ServiceHost)

	if err != nil {
		log.Fatal().Str("serviceHost", handle.ServiceHost).Err(err).Msg("Failed looking up ipv4 address of service hostname " + errorSuffix)
	}

	// Get the port of the SAML server for our password.
	u, _ := url.Parse("http://" + handle.Config.Server.Addr)

	// Save auth file for openvpn
	tmpAuthConifg, err := saveOpenVPNAuthConfig(handle.TempDir, "ACS::"+u.Port())

	if err != nil {
		log.Fatal().Err(err).Msg("Failed saving openvpn auth config file! " + errorSuffix)
	}

	log.Info().
		Str("config", handle.OpenVPNConnectionConfig.Filename).
		Str("remote", handle.ServiceIPv4).
		Msg("Fetching redirect URL from service...")

	command := exec.Command(
		handle.Config.Vpn.OpenVPN,
		"--verb", "3",
		"--config", handle.OpenVPNConnectionConfig.Filename,
		"--proto", handle.OpenVPNConnectionConfig.Protocol,
		"--remote", handle.ServiceIPv4, strconv.FormatInt(int64(handle.OpenVPNConnectionConfig.Port), 10),
		"--auth-user-pass", tmpAuthConifg,
	)

	out, err := command.CombinedOutput()

	removeErr := os.Remove(tmpAuthConifg)

	if removeErr != nil {
		log.Warn().Str("openvpnAuthConfig", tmpAuthConifg).Err(err).Msg("Failed deleting tmp openvpn auth config! " + errorSuffix)
	}

	log.Debug().Str("command", command.String()).Str("payload", string(out)).Msg("Executed command")

	// Now are must extract the URL from the payload. We use xurls to do this since regex is hard.
	rxStrict := xurls.Strict()
	foundURLs := rxStrict.FindAllString(string(out), -1)

	if len(foundURLs) == 0 {
		log.Fatal().Err(err).Msg("No URLs found in payload from server! Please check the DEBUG logs for more information. " + errorSuffix)
	}

	if len(foundURLs) > 1 {
		log.Fatal().Strs("foundURLs", foundURLs).Msg("More then one URL found in response payload! " + errorSuffix)
	}

	authUrl := foundURLs[len(foundURLs)-1]

	log.Info().Msgf("open to authenticate into OpenVPN tunnel: %s", authUrl)

	if handle.Config.Browser {
		errOpenDefaultBrowser := openDefaultBrowser(handle.Config.Vpn.User, authUrl)

		if errOpenDefaultBrowser != nil {
			log.Warn().Err(err).Msg("Failed opening default browser. Please use the provided link in the output")
		}
	}

	log.Info().Msg("Waiting for SAML response from 3rd party service...")
	SAMLResponse := <-handle.SAMLResponse

	log.Info().Msg("Received SAML response! Attempting to start OpenVPN client tunnel...")

	// Save auth file for openvpn
	SID, err := extractSIDFromOpenVPN(string(out))

	if err != nil {
		log.Fatal().Err(err).Msg("Failed finding SID in initial handshake! Please enable DEBUG mode to see payload. " + errorSuffix)
	}

	escapedSAMLResponse := url.QueryEscape(SAMLResponse)
	log.Debug().Str("SAMLResponse", escapedSAMLResponse).Msgf("writing temp openvpn auth file")
	tmpAuthConifg, err = saveOpenVPNAuthConfig(handle.TempDir, "CRV1::"+SID+"::"+escapedSAMLResponse)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed saving auth config for OpenVPN tunnel! " + errorSuffix)
	}

	baseCommand := exec.Command(
		handle.Config.Vpn.OpenVPN,
		"--verb", "3",
		"--config", handle.OpenVPNConnectionConfig.Filename,
		"--proto", handle.OpenVPNConnectionConfig.Protocol,
		"--remote", handle.ServiceIPv4, strconv.FormatInt(int64(handle.OpenVPNConnectionConfig.Port), 10),
		"--script-security", "2",
		"--auth-user-pass", tmpAuthConifg,
	)

	// If the user didn't provide a shell or we are already running as root.
	// Non special hacks are needed to to run this step.
	if handle.Config.Vpn.Shell == "" || isRoot() {
		baseCommand.Env = os.Environ()
		baseCommand.Stdout = os.Stdout
		baseCommand.Stderr = os.Stderr
		baseCommand.Stdin = os.Stdin

		log.Debug().Str("command", baseCommand.String()).Msg("Executing OpenVPN tunnel.")

		err = baseCommand.Start()
		baseCommand.Wait()
	} else {

		args := append(handle.Config.Vpn.ShellArgs, handle.Config.Vpn.Sudo+" "+baseCommand.String())

		shellCommand := exec.Command(
			handle.Config.Vpn.Shell,
			args...,
		)

		shellCommand.Env = os.Environ()
		shellCommand.Stdout = os.Stdout
		shellCommand.Stderr = os.Stderr
		shellCommand.Stdin = os.Stdin

		log.Debug().Str("command", shellCommand.String()).Msg("Executing OpenVPN tunnel in shell.")

		err = shellCommand.Start()

		shellCommand.Wait()
	}

	if err != nil {
		log.Fatal().Err(err).Msg("Failed starting OpenVPN tunnel! " + errorSuffix)
	}
}

func startSAMLServer(handle *serveHandle) {
	http.HandleFunc("/", SAMLServer(handle))
	http.ListenAndServe(handle.Config.Server.Addr, nil)
}

func writeEmbededHtmlFile(file embed.FS, filePath string, w http.ResponseWriter) {
	content, err := file.ReadFile(filePath)
	if err != nil {
		log.Error().Msgf("failed loading HTML file: %s", filePath)
		http.Error(w, "Could not load HTML file", http.StatusInternalServerError)
		return
	}

	w.Write(content)
}

func SAMLServer(handle *serveHandle) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "text/html")
		switch r.Method {
		case "POST":
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				writeEmbededHtmlFile(errorHtmlFile, "html/error.html", w)
				log.Error().Err(err).Msg("ParseForm() returned unexpected error")
				return
			}

			SAMLResponse := r.FormValue("SAMLResponse")
			if len(SAMLResponse) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				writeEmbededHtmlFile(errorHtmlFile, "html/error.html", w)
				log.Error().Msg("SAMLResponse field empty")
				return
			}

			handle.SAMLResponse <- SAMLResponse
			writeEmbededHtmlFile(welcomeHtmlFile, "html/index.html", w)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			writeEmbededHtmlFile(errorHtmlFile, "html/error.html", w)
			log.Error().Msgf("Error: POST method expected, %s received", r.Method)
		}
	}
}
