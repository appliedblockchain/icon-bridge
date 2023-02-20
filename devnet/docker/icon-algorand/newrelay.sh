rm -r cache
mkdir cache

cd ../../../pyteal
./build.sh bmc.bmc bmc
./build.sh bsh.bsh bsh

cd ../devnet/docker/goloop

docker build -t icon-algorand_goloop .
docker run -d \
    --name goloop \
    -p 9080:9080 \
    -e GOLOOP_NODE_DIR=/goloop/data/goloop \
    -e GOLOOP_LOG_WRITER_FILENAME=/goloop/data/log/goloop.log \
    -t icon-algorand_goloop

#sudo mkdir /tmp/algorand
#sudo wget -O /tmp/algorand/algorand.tar.gz https://github.com/algorand/go-algorand/releases/download/v3.13.3-stable/node_stable_darwin-amd64_3.13.3.tar.gz
##cd /tmp/algorand
#sudo tar xf algorand.tar.gz
#cd bin
#sudo mv algod goal kmd /usr/local/bin

cd /Users/pedrolino/repos/icon-bridge/devnet/algorand

goal network create -r /tmp/testnet -t ./template.json

cp ./config.json /tmp/testnet/Node
cp ./algod.token /tmp/testnet/Node
cp ./kmd_config.json /tmp/testnet/Node/kmd-v0.5/kmd_config.json
cp ./kmd.token /tmp/testnet/Node/kmd-v0.5/kmd.token

goal network start -r /tmp/testnet

cd ../../cmd/tools/algorand
./install-tools.sh

cd ../../../devnet/docker/icon-algorand

docker exec goloop goloop chain ls | jq -r '.[0] | .nid' >cache/nid

if [ ! -f goloop.keystore.json ]; then
    goloop ks gen --out icon.keystore.json
    KS_ADDRESS=$(cat icon.keystore.json | jq -r '.address')
    PASSWORD=$(docker exec goloop cat goloop.keysecret)
    docker exec goloop goloop rpc sendtx transfer \
        --uri http://localhost:9080/api/v3 \
        --nid $(cat cache/nid) \
        --step_limit=3000000000 \
        --key_store goloop.keystore.json --key_password $PASSWORD \
        --to $KS_ADDRESS --value=2001
fi

CONTRACT=../../../javascore/bmc/build/libs/bmc-optimized.jar

DEPLOY_TXN_ID=$(goloop rpc sendtx deploy $CONTRACT \
    --uri http://localhost:9080/api/v3 \
    --key_store ./icon.keystore.json --key_password gochain \
    --nid $(cat cache/nid) \
    --content_type application/java \
    --step_limit 3000000000 \
    --param _net="$(cat cache/nid).icon")

./../../algorand/scripts/wait_for_transaction.sh $DEPLOY_TXN_ID scoreAddress \
    >cache/icon_bmc_addr

CONTRACT=../../../javascore/dummyBSH/build/libs/dummyBSH-optimized.jar

TXN_ID=$(
    goloop rpc sendtx deploy $CONTRACT \
        --uri http://localhost:9080/api/v3 \
        --key_store icon.keystore.json --key_password gochain \
        --nid $(cat cache/nid) --step_limit 10000000000 \
        --content_type application/java
)

./../../algorand/scripts/wait_for_transaction.sh $TXN_ID scoreAddress >cache/icon_dbsh_addr

msg=$(
    goloop rpc call --to $(echo $(cat cache/icon_dbsh_addr) | cut -d '"' -f 2) \
        --uri http://localhost:9080/api/v3 \
        --method getLastReceivedMessage
)
echo -n $msg | xxd -r -p

sleep 1

TXN_ID=$(goloop rpc sendtx call --to $(cat cache/icon_bmc_addr) \
    --method addService \
    --value 0 \
    --param _addr=$(cat cache/icon_dbsh_addr) \
    --param _svc="dbsh" \
    --step_limit=3000000000 \
    --uri http://localhost:9080/api/v3 \
    --key_store icon.keystore.json --key_password gochain \
    --nid=$(cat cache/nid))

./../../algorand/scripts/wait_for_transaction.sh $TXN_ID

export PATH=$PATH:~/go/bin
ALGOD_ADDRESS=http://localhost:4001
ALGOD_TOKEN=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
KMD_ADDRESS=http://localhost:4002
KMD_TOKEN=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa

kmd -d /tmp/testnet/Node/kmd-v0.5 &
sleep 5

PRIVATE_KEY=$(KMD_ADDRESS=$KMD_ADDRESS KMD_TOKEN=$KMD_TOKEN kmd-extract-private-key 1)
BMC_TX_ID=$(PRIVATE_KEY=$PRIVATE_KEY ALGOD_ADDRESS=$ALGOD_ADDRESS ALGOD_TOKEN=$ALGOD_TOKEN deploy-contract ../../../pyteal/teal/bmc)
DUMMY_BSH_TX_ID=$(PRIVATE_KEY=$PRIVATE_KEY ALGOD_ADDRESS=$ALGOD_ADDRESS ALGOD_TOKEN=$ALGOD_TOKEN deploy-contract ../../../pyteal/teal/bsh)
BMC_APP_ID=$(ALGOD_ADDRESS=$ALGOD_ADDRESS ALGOD_TOKEN=$ALGOD_TOKEN get-app-id $BMC_TX_ID)
DUMMY_BSH_APP_ID=$(ALGOD_ADDRESS=$ALGOD_ADDRESS ALGOD_TOKEN=$ALGOD_TOKEN get-app-id $DUMMY_BSH_TX_ID)

echo $DUMMY_BSH_APP_ID

echo $PRIVATE_KEY >cache/algo_private_key
printf '%s' "$BMC_APP_ID" >cache/bmc_app_id
printf '%s' "$DUMMY_BSH_APP_ID" >cache/dbsh_app_id

goal app info --app-id $BMC_APP_ID -d /tmp/testnet/Node |
    awk -F ':' '/Application account:/ {gsub(/^[[:space:]]+|[[:space:]]+$/,"",$2); \
          print "btp://0x14.algo/" $2}' >cache/algo_btp_addr


LINK_TXN_ID=$(goloop rpc sendtx call --to $(cat cache/icon_bmc_addr) \
    --method addLink --param _link=$(cat cache/algo_btp_addr) \
    --key_store ./icon.keystore.json --key_password gochain \
    --nid $(cat cache/nid) --step_limit 3000000000 --uri http://localhost:9080/api/v3)

./../../algorand/scripts/wait_for_transaction.sh $LINK_TXN_ID

goal node lastround -d /tmp/testnet/Node >cache/algo_last_round

ADD_ROUND=$(goloop rpc sendtx call --to $(cat cache/icon_bmc_addr) --method setLinkRxHeight \
    --param _link=$(cat cache/algo_btp_addr) --param _height=$(cat cache/algo_last_round) \
    --key_store ./icon.keystore.json --key_password gochain \
    --nid $(cat cache/nid) --step_limit 3000000000 --uri http://localhost:9080/api/v3)

./../../algorand/scripts/wait_for_transaction.sh $ADD_ROUND

ADD_RELAY=$(goloop rpc sendtx call --to $(cat cache/icon_bmc_addr) --method addRelay \
    --param _link=$(cat cache/algo_btp_addr) \
    --param _addr=$(cat icon.keystore.json | jq .address) \
    --key_store ./icon.keystore.json --key_password gochain \
    --nid $(cat cache/nid) --step_limit 3000000000 --uri http://localhost:9080/api/v3)

./../../algorand/scripts/wait_for_transaction.sh $ADD_RELAY

cd ../../../cmd/iconbridge

ICON_ALGO="../../devnet/docker/icon-algorand"
ICON_BTP="btp://$(cat "$ICON_ALGO/cache/nid").icon/$(cat "$ICON_ALGO/cache/icon_bmc_addr")"
echo $ICON_BTP >$ICON_ALGO/cache/icon_btp_addr
ALGO_BTP=$(cat "$ICON_ALGO/cache/algo_btp_addr")
BMC_ID=$(cat "$ICON_ALGO/cache/bmc_app_id")
ALGO_ROUND=$(cat "$ICON_ALGO/cache/algo_last_round" | tr -d '\n')

ALGOD_ADDRESS=$ALGOD_ADDRESS ALGOD_TOKEN=$ALGOD_TOKEN PRIVATE_KEY=$PRIVATE_KEY register-dummy-bsh ../../pyteal/teal

jq --arg ICON_BTP "$ICON_BTP" --arg ALGO_BTP "$ALGO_BTP" --argjson ALGO_ROUND "$ALGO_ROUND" --argjson BMC_ID "$BMC_ID" '.relays[0].dst.address=$ICON_BTP | .relays[0].src.address=$ALGO_BTP | .relays[0].src.options.verifier.round=$ALGO_ROUND | .relays[0].src.options.appID=$BMC_ID' $ICON_ALGO/algo-config.json > a2i_tmp.json

mv a2i_tmp.json $ICON_ALGO/algo-config.json

go run . -config=$ICON_ALGO/algo-config.json
