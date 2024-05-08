alias autocctpd=./simapp/build/simd

for arg in "$@"
do
    case $arg in
        -r|--reset)
        rm -rf .autocctp
        shift
        ;;
    esac
done

if ! [ -f .autocctp/data/priv_validator_state.json ]; then
  autocctpd init validator --chain-id "autocctp-1" --home .autocctp &> /dev/null

  autocctpd keys add validator --home .autocctp --keyring-backend test &> /dev/null
  autocctpd genesis add-genesis-account validator 1000000ustake --home .autocctp --keyring-backend test
  autocctpd keys add user --home .autocctp --keyring-backend test &> /dev/null
  autocctpd genesis add-genesis-account user 10000000uusdc --home .autocctp --keyring-backend test

  TEMP=.autocctp/genesis.json
  touch $TEMP && jq '.app_state.staking.params.bond_denom = "ustake"' .autocctp/config/genesis.json > $TEMP && mv $TEMP .autocctp/config/genesis.json

  autocctpd genesis gentx validator 1000000ustake --chain-id "autocctp-1" --home .autocctp --keyring-backend test &> /dev/null
  autocctpd genesis collect-gentxs --home .autocctp &> /dev/null

  sed -i '' 's/timeout_commit = "5s"/timeout_commit = "1s"/g' .autocctp/config/config.toml
fi

autocctpd start --home .autocctp
