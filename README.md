
1.
```bash
cd funny

go build -o dns

```

2.
```bash
dns -f=dns/funny/dns.json -log_dir=dns/funny/log

dig @127.0.0.1 www.github.com
```