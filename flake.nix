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

      # Build a single-arch OCI image tarball. `arch` is Docker naming
      # (amd64/arm64); `crossPkgs` is the pkgsCross set that produces
      # the binary for that arch.
      mkImage = pkgs: arch: crossPkgs: pkgs.dockerTools.buildLayeredImage {
        name = "particulate";
        tag = arch;
        architecture = arch;
        contents = [ (mkParticulate crossPkgs) ];
        config = {
          Cmd = [ "/bin/particulate" ];
          ExposedPorts."8000/tcp" = { };
        };
      };
    in {
      packages = forAllSystems ({ system, pkgs }: {
        # Native binary for local dev.
        default = mkParticulate pkgs;

        # Per-arch OCI image tarballs. Publish each with a platform tag,
        # then combine into a manifest list at :latest in CI.
        dockerAmd64 = mkImage pkgs "amd64" pkgs.pkgsCross.gnu64;
        dockerArm64 = mkImage pkgs "arm64" pkgs.pkgsCross.aarch64-multiplatform;
      });

      devShells = forAllSystems ({ pkgs, ... }: {
        default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools skopeo manifest-tool ];
        };
      });
    };
}
