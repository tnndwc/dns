
1.
```bash
cd funny

go build -o dns

```

2.
```bash
dns -h 127.0.0.1 -p 53  -cf dns/funny/

dig @127.0.0.1 www.github.com
```