# simplefileserver
A very simple file server written in Go for file and folder downloading through a browser.
Folders will be streamed to the browser as an uncompressed .zip file (requires no extra server hard drive space).

## Usage
```
$ go build main.go
$ ./main [-port 8090] [-cert ./certfile.cert] [-key ./keyfile.key] [-root /path/to/root/dir/]
```


## Notes
* Certs most easily obtained from letsencrypt.
* To generate temporary TLS key and cert files:
```
$ openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
$ openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
```
* Only supports Linux (works on WSL).

## Todo
* Do actual web design.
* Generate temporary share links (no login required).
