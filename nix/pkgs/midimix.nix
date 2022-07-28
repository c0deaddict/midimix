{ lib, buildGoModule, alsaLib }:

buildGoModule rec {
  pname = "midimix";
  version = "0.0.1";

  src = ../..;

  # go mod strips directories without packages. This strips parts of the gomidi
  # rtmidi cpp code.
  proxyVendor = true;
  vendorSha256 = "sha256-RVfx9+1KSfYYyeOX8V2O4zH9KWEiJ+V2GtGCQxE2A9c=";

  subPackages = [ "cmd/midimix" ];

  buildInputs = [ alsaLib ]; # for rtmidi

  meta = with lib; {
    description = "AKAI MIDIMix control";
    homepage = "https://github.com/c0deaddict/midimix";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
