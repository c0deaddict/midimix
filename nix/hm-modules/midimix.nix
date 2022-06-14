{ pkgs, lib, config, ... }:

with lib;

let

  cfg = config.services.midimix;
  format = pkgs.formats.json { };
  configFile = format.generate "config.json" cfg.settings;

  audioService =
    if cfg.pipewire then "pipewire-pulse.service" else "pulseaudio.service";

in {

  options.services.midimix = {
    enable = mkEnableOption "midimix";

    pipewire = mkOption {
      type = types.bool;
      default = false;
    };

    package = mkOption {
      type = types.package;
      default = pkgs.callPackage ../pkgs/midimix.nix {};
    };

    settings = mkOption {
      default = { };
      type = format.type;
    };
  };

  config = mkIf cfg.enable {
    systemd.user.services.midimix = {
      Unit = {
        Description = "MIDIMix control panel";
        After = [ audioService ];
        Requires = [ audioService ];
      };

      Service = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/midimix -config ${configFile}";
        Environment = "GOMAXPROCS=1";
        Restart = "on-failure";
        RestartSec = 3;
      };

      Install = { WantedBy = [ "graphical-session.target" ]; };
    };
  };
}
