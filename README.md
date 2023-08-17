# rmfakecloud-proxy
Single-minded HTTPS reverse proxy -> repurposed as a domain-targeted network logger

(forked from https://github.com/yi-jiayu/secure)

Log xochitl traffic.

    journalctl -u proxy -f

Note: Not all traffic is logged, only certain domain names.
If you are looking to add domain names, you must

1. identify the domain names (wireshark?)
1. add them to installer.sh (both `[ san ]` and `/etc/hosts`)
1. recompile and reinstall

## Usage
```
usage: rmfake-proxy -c [config.yml] [-addr host:port] -cert certfile -key keyfile [-version]
  -addr string
        listen address (default ":443")
  -c string
        config file
  -cert string
        path to cert file
  -key string
        path to key file
  -version
        print version string and exit
```

### Example
```
rmfake-proxy -cert cert.pem -key key.pem
```

## Configfile
```yaml
cert: proxy.crt
key: proxy.key
#addr: :443
```
