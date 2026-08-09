#!/bin/bash

ip="202.122.49.242"
port=62425

eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_rsa

# Останавливаем screen-сессии
ssh -p "$port" root@"$ip" "screen -S arb -X quit 2>/dev/null || true"
ssh -p "$port" root@"$ip" "screen -S ntf -X quit 2>/dev/null || true"

# На всякий случай убиваем процессы по имени (если screen уже умер, а процесс остался)
ssh -p "$port" root@"$ip" "pkill -f /home/arb/arb-bot 2>/dev/null || true"
ssh -p "$port" root@"$ip" "pkill -f /home/notifier/tgn 2>/dev/null || true"

# Удаляем папки
ssh -p "$port" root@"$ip" "rm -rf /home/arb /home/notifier"

ssh-agent -k

echo "Stopped and cleaned."