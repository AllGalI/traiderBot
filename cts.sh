
ip="202.122.49.242"
port=62425
go build -o arb-bot
scp -P "$port" ./arb-bot root@"$ip":/home/arb/