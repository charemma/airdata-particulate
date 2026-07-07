{
  description = "SDS011 particulate matter Prometheus exporter";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      forAllSystems = f: nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ] (system: f {
        inherit system;
        pkgs = nixpkgs.legacyPackages.${system};
      });

      mkParticulate = pkgs: pkgs.buildGoModule {
        pname = "particulate";
        version = "0.1.0";
        src = ./.;
        vendorHash = "sha256-VlvXm3BigopGsFPl+Et3nWeMhWG3h+883bbdMeWe2Oo=";
        env.CGO_ENABLED = "0";
        subPackages = [ "." ];
        meta.mainProgram = "particulate";
      };
    in {
      packages = forAllSystems ({ system, pkgs }: {
        # Native binary for local dev / testing on this host's arch.
        default = mkParticulate pkgs;

        # OCI image built for arm64 (aiagent target). Cross-compiled from
        # whichever host runs `nix build .#docker`; publish with skopeo.
        docker = pkgs.dockerTools.buildLayeredImage {
          name = "airdata-particulate";
          tag = "latest";
          architecture = "arm64";
          contents = [ (mkParticulate pkgs.pkgsCross.aarch64-multiplatform) ];
          config = {
            Cmd = [ "/bin/particulate" ];
            ExposedPorts."8000/tcp" = { };
          };
        };
      });

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools skopeo ];
        };
      });
    };
}
