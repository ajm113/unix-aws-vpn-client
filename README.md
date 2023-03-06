# aws-vpn-client

This is PoC to connect to the AWS Client VPN with OSS OpenVPN using SAML
authentication. Tested on macOS and Linux, should also work on other POSIX OS with a minor changes.

See [my blog post](https://smallhacks.wordpress.com/2020/07/08/aws-client-vpn-internals/) for the implementation details.

P.S. Recently [AWS released Linux desktop client](https://aws.amazon.com/about-aws/whats-new/2021/06/aws-client-vpn-launches-desktop-client-for-linux/), however, it is currently available only for Ubuntu, using Mono and is closed source. 

## Content of the repository

- [openvpn-v2.4.9-aws.patch](openvpn-v2.4.9-aws.patch) - patch required to build
AWS compatible OpenVPN v2.4.9, based on the
[AWS source code](https://amazon-source-code-downloads.s3.amazonaws.com/aws/clientvpn/osx-v1.2.5/openvpn-2.4.5-aws-2.tar.gz) (thanks to @heprotecbuthealsoattac) for the link.

## How to use

1. Build patched openvpn version and put it to the folder with a script
2. Build aws-vpn-client wrapper `go build .`
3. `cp ./awsvpnclient.yml.example ./awsvpnclient.yml` and update the necessary paths.
4. Finally run `./aws-vpn-client serve --config myconfig.openvpn` to connect to the AWS.

## Security

OpenVPN recommends running the openvpn binary as an unprivileged user after initialization (see https://openvpn.net/community-resources/hardening-openvpn-security/). The `awsvpnclinet.yml` file includes the `user` and `group` keys, demonstrating how to run
`openvpn` as the `nobody` user (and group). If those keys are not present, the binary will run continue to run as whichever
user launched it originally.

## Todo

* Unit tests
* General Code Cleanup
* Better integrate SAML HTTP server with a script or rewrite everything on golang

# Using via Nix Flakes

This program can be run via `nix`, using the `flakes` feature. You will need to know how to install nix and what flakes 
are in order to follow these instructions.

## Apps

Two apps are defined. One makes it easy to open a tunnel with a given VPN profile, the other lets you run the original program (meaning
you must provide all arguments):

- *default app* - Use `nix run .` (or replace `.` with the flake reference for this repo) to run the default program. Just give a path to the OpenVPN configuration file and it should work. Note you will likely
need to run under `sudo`:

```
$ sudo su
...
# nix run . -- ~/.config/AWSVPNClient/OpenVpnConfigs/<profile>
```

Note that this app is hard-coded to run as the `nobody` user (and group). If that does not exist on your system, you will have
to override the existing configuration.

- *aws-vpn-client-unwrapped app* - Use `nix run .#aws-vpn-client-unwrapped` to run the original program, allowing more control over arguments given. 

## Packages

This flake provides two main packages, `aws-vpn-client` (also the default package) and `aws-vpn-client-unwrapped`. 

Besides those two packages, it also provides a patched `openvpn` client (necessary to using this program).

### `aws-vpn-client-unwrapped`

This is the original program from this repo, provided for more control over arguments. For convenience, a `awsvpnclient.yml` is generated when the program is installed and is placed 
in the `bin` directory next to the executable. (It will not be used automatically tho - the original program always looks in the current workign directory or
your home directory for that file). 
### `aws-vpn-client` 

This is a wrapper around the original program, updated so you can just pass the path to a VPN configuration and it will open that tunnel. 

## Shell (Development)

This flake uses the excellent tools from `devensh.sh` to provide a Go environment for development. Use `nix develop` to
enter the shell.