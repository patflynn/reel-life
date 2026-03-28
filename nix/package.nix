{ lib, buildGoModule }:

buildGoModule {
  pname = "reel-life";
  version = "0.1.0";
  src = ./..;

  vendorHash = "sha256-BH08qUCFyYWfPkO49ve0lTIUhfdSbnR9ZE8dVULtgCU=";

  subPackages = [ "cmd/reel-life" ];

  meta = {
    description = "AI-powered chatops agent for media curation";
    mainProgram = "reel-life";
  };
}
