{
  description =
    "Go application for controlling PulseAudio and LED installations with a MIDI panel";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable"; };

  outputs = inputs@{ self, nixpkgs, ... }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in {
      overlay = final: prev: import ./nix/pkgs/default.nix { pkgs = final; };

      hmModules.midimix = import ./nix/hm-modules/midimix.nix;
      hmModule = self.nixosModules.midimix;

      packages = forAllSystems (system:
        import ./nix/pkgs/default.nix rec {
          pkgs = import nixpkgs { inherit system; };
        });
      defaultPackage = forAllSystems (system: self.packages.${system}.midimix);
    };
}
