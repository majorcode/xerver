xerver v3.1
============
A transparent blazing fast fastcgi reverse proxy .

Features
============
* Cross platform
* Accelerated and optimized without module hell
* No configuration needed  
* Standalone, Tiny & Lightweight
* Supports both http and https
* Automatically use HTTP/2 (in https)

How It Works
=============
* A request hits the `xerver`
* `xerver` handles the request
* `xserver` sends it to the backend `fastcgi` process' main controller file
* The controller does its part
* `fastcgi` process replies to `xerver` with the results
* `xerver` parses the result sends it to the client

Building from source
==================
1- make sure you have `Golang` installed .  
2- `go get -u github.com/majorcode/xerver`  
3- `go install github.com/majorcode/xerver`  
4- make sure `$GOPATH` in your `$PATH` env var .    
5- `xerver --help`

Example (1)
==============
**Only acts as a static file server by default on port 80**
```bash
xerver --root=/path/to/www/ --http=:80
```

Example (2)
==============
**Listen on address `0.0.0.0:80`** and send the requests to `./controller.php`  
```bash
xerver --backend=unix:/var/run/php5-fpm.sock controller=./controller.php --http=:80
```
** OR Listen on address `0.0.0.0:80` & ``0.0.0.0:443`` ** and send the requests to `./controller.php`
```bash
xerver --backend=unix:/var/run/php5-fpm.sock controller=./controller.php --http=:80 --https=:443 --cert=./cert.pem --key=./key.pem
```


Authors
==================
Created By [Mohammed Al Ashaal](http://www.alash3al.xyz)

This version: [Stephen Farmer](http://www.majorcode.com)
