osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort 3001 -gossipAddr 127.0.0.1:4001 -peers 127.0.0.1:4002 -name A -rtimer 100 -hw3ex3=true -N=4"'
osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort 3002 -gossipAddr 127.0.0.1:4002 -peers 127.0.0.1:4003 -name B -rtimer 100 -hw3ex3=true -N=4"'
osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort 3003 -gossipAddr 127.0.0.1:4003 -peers 127.0.0.1:4004 -name C -rtimer 100 -hw3ex3=true -N=4"'
osascript -e 'tell app "Terminal" to do script "cd ~/go/src/github.com/vquelque/Peerster/ && ./Peerster -UIPort 3004 -gossipAddr 127.0.0.1:4004 -peers 127.0.0.1:4001 -name D -rtimer 100 -hw3ex3=true -N=4"'



