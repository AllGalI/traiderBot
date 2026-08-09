#!/bin/bash

ip="202.122.49.242"
port=62425

eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa

# Сборка
cd /home/ali/goProjects/tgNotifier/
go build -o tgn

cd /home/ali/goProjects/crypto-monitor
go build -o arb-bot

# Копирование
ssh -p "$port" root@"$ip" "mkdir -p /home/arb /home/notifier"
scp -P "$port" /home/ali/goProjects/tgNotifier/tgn root@"$ip":/home/notifier/
scp -P "$port" /home/ali/goProjects/crypto-monitor/arb-bot root@"$ip":/home/arb/

# Запуск в detached screen-сессиях
ssh -p "$port" root@"$ip" "apt install screen"
ssh -p "$port" root@"$ip" "screen -dmS arb bash -c 'cd /home/arb && ./arb-bot'"
ssh -p "$port" root@"$ip" "screen -dmS ntf bash -c 'cd /home/notifier && ./tgn --data=/home/arb/stats.log'"

ssh-agent -k