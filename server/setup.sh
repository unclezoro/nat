docker run -t --name vm1 --rm --network  internal1 --privileged=true -v /Users/baifudong/workspace/src/awesomeProject1/server:/root/work  my-python  python -m http.server 8080&
sleep 1
docker run -t --name nat1 --rm --network internal1 --privileged=true   my-python  python -m http.server 6060 &
sleep 1
docker network connect public nat1
docker run -t --name vm2 --rm --network  internal2  --privileged=true -v /Users/baifudong/workspace/src/awesomeProject1/server:/root/work  my-python python -m http.server 8080 &
sleep 1
docker run -t --name nat2 --rm --network internal2 --privileged=true   my-python  python -m http.server 6060 &
sleep 1
docker network connect public nat2
sleep 1
docker run -t --name pubserver --rm --network  public -v /Users/baifudong/workspace/src/awesomeProject1/server:/root/work  my-python  python -m http.server 8080 &
sleep 2

docker exec -t vm2 route add default gw  172.21.0.3
docker exec -t vm1 route add default gw  172.20.0.3

docker exec -t nat1  sysctl -p
docker exec -t nat1  iptables -A FORWARD -p tcp -j ACCEPT
docker exec -t nat1  iptables -t nat -A POSTROUTING -o eth1 -j SNAT --to-source "172.22.0.2"
docker exec -t nat1  iptables -t nat -A PREROUTING -i eth1  -j DNAT --to-destination "172.20.0.2:8080"

docker exec -t nat2  sysctl -p
docker exec -t nat2  iptables -A FORWARD -p tcp -j ACCEPT
docker exec -t nat2  iptables -t nat -A POSTROUTING -o eth1 -j SNAT --to-source "172.22.0.3"
docker exec -t nat2  iptables -t nat -A PREROUTING -i eth1  -j DNAT --to-destination "172.21.0.2:8080"

sleep 2

docker exec -it  vm1 curl  172.22.0.4:8080
docker exec -it  vm2 curl  172.22.0.4:8080
docker exec -it  pubserver curl  172.22.0.3:8080
