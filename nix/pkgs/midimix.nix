{ lib, buildGoModule, alsa-lib }:

buildGoModule rec {
  pname = "midimix";
  version = "0.0.1";

  src = ../..;

  # go mod strips directories without packages. This strips parts of the gomidi
  # rtmidi cpp code.
  proxyVendor = true;
  vendorHash = "sha256-FNDwwrvGPzjFQaVfCClW/ODYQ9G5N+P1ky0KoUQb/JY=";

  subPackages = [ "cmd/midimix" ];

  buildInputs = [ alsa-lib ]; # for rtmidi

  meta = with lib; {
    description = "AKAI MIDIMix control";
    homepage = "https://github.com/c0deaddict/midimix";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
