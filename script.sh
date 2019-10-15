osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort=3001 -gossipAddr=127.0.0.1:4001 -name=P1 -peers=127.0.0.1:4002 -uisrv t"'
osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort=3002 -gossipAddr=127.0.0.1:4002 -name=P2 -peers=127.0.0.1:4003 -uisrv t"'
osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort=3003 -gossipAddr=127.0.0.1:4003 -name=P3 -uisrv t"'
