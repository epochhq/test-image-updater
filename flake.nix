{
  description = "test-image-updater Go toolchain";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/1e8a2e6b08f0940fc0d94aeff0e3e940ee86b632";
    nixpkgs-go.url = "github:NixOS/nixpkgs/be147c05f4a91a83e254e9ce5d7f0421f99fca5c";
    nixpkgs-intel.url = "github:NixOS/nixpkgs/da5ad661ba4e5ef59ba743f0d112cbc30e474f32";
    go-source.url = "file+https://go.dev/dl/go1.27.3.src.tar.gz";
    go-source.flake = false;
  };
  outputs = inputs@{ nixpkgs, nixpkgs-intel, nixpkgs-go, ... }:
    let
      lock = builtins.fromJSON (builtins.readFile ./flake.lock);
      source = lock.nodes.${lock.nodes.root.inputs.go-source};
      version = builtins.elemAt (builtins.match "https://go[.]dev/dl/go([0-9]+[.][0-9]+[.][0-9]+)[.]src[.]tar[.]gz" source.original.url) 0;
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forSystems = f: builtins.listToAttrs (map (system: { name = system; value = f system; }) systems);
      packages = forSystems (system:
        let
          pkgs = import (if system == "x86_64-darwin" then nixpkgs-intel else nixpkgs) { inherit system; };
          selectedGo = if system == "x86_64-darwin"
            then pkgs.callPackage (nixpkgs-go + "/pkgs/development/compilers/go/1.27.nix") {
              buildGo127Module = pkgs.buildGoModule.override { go = selectedGo; };
            }
            else (import nixpkgs-go { inherit system; }).go_1_27;
          go = assert selectedGo.version == version;
            assert selectedGo.src.url == source.locked.url;
            selectedGo;
        in { inherit go pkgs; }
      );
    in assert source.locked.url == source.original.url;
    {
      runtimeConsistency = "Go ${version} compiler source matches its native lock\n";
      packages = forSystems (system: { inherit (packages.${system}) go; });
      checks = forSystems (system:
        let inherit (packages.${system}) pkgs go;
        in {
          toolchain = pkgs.runCommand "go-module-toolchain-consistency" { nativeBuildInputs = [ go ]; } ''
            export GOTOOLCHAIN=local GOWORK=off GOPROXY=off GOSUMDB=off
            export GOCACHE="$TMPDIR/go-cache"
            mkdir module
            cd module
            cp ${./go.mod} go.mod
            mkdir devtools
            cp ${./devtools/go.mod} devtools/go.mod
            expected="$(go list -m -f '{{.GoVersion}}')"
            test "$(go env GOVERSION)" = "go$expected"
            test "$(go list -modfile=devtools/go.mod -m -f '{{.GoVersion}}')" = "$expected"
            touch $out
          '';
        });
      devShells = forSystems (system:
        let inherit (packages.${system}) pkgs go;
        in { default = pkgs.mkShell {
          packages = [ go ];
          GOTOOLCHAIN = "local";
          GOWORK = "off";
        }; });
    };
}
