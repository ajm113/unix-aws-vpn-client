# Unix AWS VPN Client

*Connects to AWS Client VPN with OSS OpenVPN using SAML authentication for Unix enviroments. Linx, Mac, BSD. Forked from @samm-git's aws-vpn-client.*

See [samm-git's blog post](https://smallhacks.wordpress.com/2020/07/08/aws-client-vpn-internals/) for the implementation details.

[AWS released Linux desktop client](https://aws.amazon.com/about-aws/whats-new/2021/06/aws-client-vpn-launches-desktop-client-for-linux/), however, it's extremely buggy and
doesn't provide useful logging. Supposedly works on Ubuntu...

## How To Build

1. Build patched openvpn version and put it to the scripts folder or somewhere of your choosing.
2. Build aws-vpn-client wrapper `go build .`
3. `cp ./awsvpnclient.yml.example ./awsvpnclient.yml` and update the necsery fields.
4. Finally run `./aws-vpn-client serve --config myconfig.openvpn` to connect to the AWS.

## Todo

* Unit tests
* Smoother user expirence running on Linux with permissions.
* Automatic script to patch and build openvpn.
