package main

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	appName      = "aws-vpn-client"
	repoUrl      = "https://github.com/ajm113/unix-aws-vpn-client"
	bugReportUrl = "https://github.com/ajm113/unix-aws-vpn-client/issues"
	errorSuffix  = "Questions? Please visit our issue tracker: " + bugReportUrl

	defaultConfigFilename      = "awsvpnclient.yml"
	defaultConfigDirectoryName = "awsvpnclient"
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = "Connects to AWS VPN service via cli without the official VPN Client hassle."
	app.EnableBashCompletion = true

	app.Commands = []*cli.Command{
		{
			Name:    "setup",
			Aliases: []string{"build"},
			Usage:   "Compiles openvpn for your unix environment and checking if baseline dependencies are installed.",
			Action:  setupAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					TakesFile: true,
					Required:  false,
					Aliases:   []string{"s"},
					Name:      "source",
					Usage:     "OpenVPN (ideally v2.5.1) source code directory. Will download source code to tmp directory by default",
				},
				&cli.StringFlag{
					TakesFile: true,
					Required:  false,
					Name:      "out",
					Aliases:   []string{"o"},
					Value:     ".",
					Usage:     "compiled openvpn binary location",
				},
				&cli.StringFlag{
					TakesFile: true,
					Required:  false,
					Name:      "patch",
					Aliases:   []string{"p"},
					Value:     "scripts/openvpn-v2.5.1-aws.patch",
					Usage:     "patch file to use against openvpn source code",
				},
			},
		},
		{
			Name:    "serve",
			Aliases: []string{"host", "start"},
			Usage:   "Loads openvpn configuration file and runs SAML server and openvpn.",
			Action:  serveAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					TakesFile: true,
					Required:  true,
					Name:      "config",
					Usage:     "raw openvpn configuration",
				},
				&cli.StringFlag{
					TakesFile: true,
					Name:      "configTmpDir",
					Value:     os.TempDir(),
					Usage:     "Temp folder location of formatted openvpn configurations.",
				},
			},
		},
	}

	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	err := app.Run(os.Args)

	if err != nil {
		log.Fatal().Err(err).Msg("closed to unexpected error")
	}
}
