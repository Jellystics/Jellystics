{
  description = "Jellystics nix direnv";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [ 
        "x86_64-linux" 
        "aarch64-darwin"
        ];
    in {
      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              nodejs_24
              pnpm
	            go-task
            ];
          };
        }
      );
    };
}
