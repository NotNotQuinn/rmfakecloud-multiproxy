# rmfakecloud multi proxy

Currently you can:
 - Log network traffic from xochitl for debugging / reversing
 - More to come (the name will fit)

## Usage
```
usage: rmfake-multiproxy -c [config.yml] [-addr host:port] -cert certfile -key keyfile [-version]
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
rmfake-multiproxy -cert cert.pem -key key.pem
```

## Configfile
```yaml
cert: proxy.crt
key: proxy.key
#addr: :443
```
