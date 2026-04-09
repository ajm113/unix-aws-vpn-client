# Unix AWS VPN Client

*Connects to AWS Client VPN with OSS OpenVPN using SAML authentication for Unix environments. Linx, Mac, BSD. Forked from @samm-git's aws-vpn-client.*

See [samm-git's blog post](https://smallhacks.wordpress.com/2020/07/08/aws-client-vpn-internals/) for the implementation details.

[AWS released Linux desktop client](https://aws.amazon.com/about-aws/whats-new/2021/06/aws-client-vpn-launches-desktop-client-for-linux/), however, it's extremely buggy and
doesn't provide useful logging. Supposedly works on Ubuntu...

## Install

*Please make sure you have Go 1.17+ installed on your system*

### Building Client

1. `$ git clone https://github.com/ajm113/unix-aws-vpn-client.git`
2. `$ cd unix-aws-vpn-client`
3. `go build .`
4. `cp ./unix-aws-vpn-client {TO a DIR that's listed in your $PATH or whatever your personal preference is}`

### Setting Up

1. Inside the root directory run `./unix-aws-vpn-client setup`.
2. Let it run until it spits out `openvpn_aws` executable. -- You may need to install required dependencies that compiler prints out if it stops.
3. Move `openvpn_aws` to a directory of your choosing.
4. Copy/paste this template into your `awsvpnclient.yml` inside `~/.config/awsvpnclient/` folder:

```yml

debug: false                            # Prints useful debugging information
browser: false                          # Opens the web browser for auth step. Works a little wonky on some distros.
vpn:
  openvpn: {path to your openvpn_aws}   # Path to openvpn_aws binary.                        
  sudo: /bin/sudo                       # Sudo command to run when establishing a tunnel to AWS.    (default is fine for most distros)
  shell: /bin/sh                        # bash/shell command when establishing a tunnel to AWS.     (default is fine for most distros)
  shellargs:                            # bash/shell commands to add when executing shell commands. (default is fine for sh)
    - "-c"
server:
  addr: "127.0.0.1:35001"              # SAML Server listen address after auth redirect. (default is fine for most setups)

```

#### How to Run OpenVPN as Non-Root (optional, but prefered!)

If you prefer to NOT give the compiled patched openvpn binary full root privilages, but still lock the executable down at a user level. 
You can easily run these shell commands:

```sh
chmod 750 /path/to/openvpn
chown root:youruser /path/to/openvpn
sudo setcap cap_net_admin+ep /path/to/openvpn_aws
```

The first to commands will lock the patched openvpn binary to your user/group.
The last command will give `CAP_NET_ADMIN` and `CAP_NET_BIND_SERVICE` minimal privileges to the patched compiled executable.
You may change the the `awsvpnclient.yaml` to have a empty string for `vpn.sudo` to avoid `sudo` login.

```yml
...
vpn:
    sudo: # leave blank
...
```

Now you can run `unix-aws-vpn-client start` without ever needing to sudo login into the patched openvpn executable!

### Running Tunnel

After everything is compiled and setup. All you have to do now is run:

```bash
$ unix-aws-vpn-client start --config myvpnfile.ovpn
```

After you successfully authenticated (and sudo login) you should now have a tunnel to AWS.

## Todo

* Unit Tests
* Cleaner Code
