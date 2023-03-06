{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-22.11";
    devenv.url = "github:cachix/devenv";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, devenv, flake-utils, ... } @ inputs:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
        };
        openvpn = pkgs.openvpn.overrideAttrs (_: { patches = [ ./scripts/openvpn-v2.5.1-aws.patch ];});
        aws-vpn-client-unwrapped = pkgs.buildGoModule {
          src = ./.;
          pname = "aws-vpn-client-unwrapped";
          version = "0.0.1";
          vendorSha256 = "sha256-602xj0ffJXQW//cQeByJjtQnU0NjqOrZWTCWLLhqMm0";

          postFixup = ''
            cat > $out/bin/awsvpnclient.yml <<-HERE
              vpn:
                openvpn: ${openvpn}/bin/openvpn
                port: 1194
                user: nobody
                group: nobody
              server:
                addr: "127.0.0.1:35001"
            HERE
            '';
        };
        aws-vpn-client = pkgs.stdenv.mkDerivation {
          pname = "aws-vpn-client";
          version = "0.0.1";
          buildInputs = [ aws-vpn-client-unwrapped ];
          nativeBuildInputs = [ pkgs.makeWrapper ];

          dontUnpack = true;
          dontPatch = true;
          dontConfigure = true;
          dontBuild = true;
          doCheck = false;

          installPhase = ''
            mkdir -p $out/bin
            makeWrapper ${aws-vpn-client-unwrapped}/bin/aws-vpn-client $out/bin/aws-vpn-client \
              --chdir ${aws-vpn-client-unwrapped}/bin \
              --add-flags "serve --config"
          '';
        };
      in {
        devShells =  {
          default = devenv.lib.mkShell {
            inherit inputs pkgs;
            modules = [
              {
                # https://devenv.sh/reference/options/
                packages = [ openvpn pkgs.inetutils pkgs.expect ];

                languages.python.enable = true;
                enterShell = ''
                  export PS1='\e[1;34mƛ > \e[0m'
                '';
              }
            ];
          };
        };
        packages = {
          inherit
            aws-vpn-client
            aws-vpn-client-unwrapped;
          openvpn = openvpn;
          default = aws-vpn-client;
        };
        apps = {
          # This runs the program with the `serve` argument and `--config` flag, so you just have
          # to provide the path to an OpenVPN config.
          #
          # Note you likely need to run this under `sudo`, as openvpn (which ultimately does the work)
          # requires root access to manage VPN tunnels.
          default = {
              type = "app";
              program = "${self.packages.${system}.aws-vpn-client}/bin/aws-vpn-client";
          };
          # This runs the bare program, so you can pass any custom arguments.
          aws-vpn-client-unwrapped = {
              type = "app";
              program = "${self.packages.${system}.aws-vpn-client-unwrapped}/bin/aws-vpn-client";
          };
        };
      }
    );
}
