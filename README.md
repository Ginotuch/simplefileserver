# simplefileserver
A very simple file server written in Go for file and folder downloading through a browser.
Folders will be streamed to the browser as an uncompressed .zip file (requires no extra server hard drive space).

Has temporary (48 hour) no login download link creation.

## Usage
```
$ go build main.go
$ ./main [-port 8090] [-cert ./certfile.cert] [-key ./keyfile.key] [-root /path/to/root/dir/]
```


## Notes
* TLS certificates most easily obtained from letsencrypt. Example arguments to use default locations for certbot:
  * `-cert /etc/letsencrypt/live/YOURSITE/fullchain.pem`
  * `-key /etc/letsencrypt/live/YOURSITE/privkey.pem`
* To generate temporary TLS key and cert files (or just run `./createTLSkeys.sh`):
```
$ openssl req  -new  -newkey rsa:2048  -nodes  -keyout localhost.key  -out localhost.csr
$ openssl  x509  -req  -days 365  -in localhost.csr  -signkey localhost.key  -out localhost.crt
```

## Todo
* Do actual web design
* Switch from http authentication to something better
* Implement database for login details storage
* Make temporary links persistent (using database)
* User uploads
