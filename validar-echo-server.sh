MSG="test_echo"
SERVER_CONTAINER="server"
SERVER_PORT=$(grep SERVER_PORT server/config.ini | awk -F'=' '{print $2}' | xargs)
SERVER_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $SERVER_CONTAINER)
RESPONSE=$(docker run --rm --network tp0_testing_net busybox sh -c "echo $MSG | nc $SERVER_IP $SERVER_PORT")

if [ "$RESPONSE" = "$MSG" ]; then
    echo "action: test_echo_server | result: success"
else
    echo "action: test_echo_server | result: fail"
fi