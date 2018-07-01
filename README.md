cd mini_dns

go build -o dns

./dns -h 127.0.0.1 -p 53  -cf mini_dns_server/

dig @127.0.0.1 www.github.com