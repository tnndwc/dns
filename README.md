cd mini_dns_server

go build -o dns

./dns -h 127.0.0.1 -p 53  -cf /tmp/

dig @127.0.0.1 www.github.com